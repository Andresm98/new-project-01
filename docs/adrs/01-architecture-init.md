# ADR-01: Arquitectura inicial — Fullstack dockerizado con fase CI

| Campo      | Valor                         |
|------------|-------------------------------|
| Autor      | Santiago Moreta               |
| Fecha      | 27 nov 2026                   |
| Estado     | Implementado (Ola 1)          |
| Alcance    | CI + CD a la EC2 (SSH)         |

---

## 1. Propósito

Generar una infraestructura que contenga la **fase CI de un pipeline dockerizado**. El entorno local se orquesta con **Docker Compose** y se compone de dos aplicaciones propias, cada una construida sobre su propia **golden image**, más **Prometheus** para el scraping de métricas y **Grafana** en contenedor para visualizarlas. El acceso externo a Grafana se restringe con una *allow list* en el **Security Group** del host.

## 2. Contexto

Se necesita una aplicación fullstack mínima pero representativa que sirva como base para:

- Demostrar arquitectura limpia/hexagonal en backend y patrones modernos en frontend.
- Exponer métricas operativas basadas en las **4 golden signals** de Google (latencia, tráfico, errores, saturación).
- Validar un flujo CI completo sobre imágenes Docker: **build → security check → approve**.

### 2.1 Componentes

**Imagen 1 — Backend (Go)**

- Aplicación backend sencilla en Go con **Clean + Hexagonal Architecture**: `domain`, `app` (casos de uso), `ports` (interfaces de entrada/salida) e `infra/adapters` (HTTP, persistencia, métricas).
- CRUD **maestro–detalle** de productos.
- Endpoint `/metrics` para scraping de **Prometheus**; como mínimo la latencia por petición (histograma).

**Imagen 2 — Frontend (Next.js)**

- Aplicación sencilla en **Next.js + React + TypeScript**.
- Comportamiento **SPA**, uso de **HoC** (Higher-Order Components) y **lazy loading** simple.
- Consume la API del backend para listar y gestionar productos.
- Estilos con **Tailwind CSS**.

**Observabilidad**

- **Prometheus** (contenedor): scrapea `/metrics` del backend. Solo accesible por la red interna del compose.
- **Grafana** (contenedor): consume el endpoint de Prometheus (`http://prometheus:9090`) como datasource y muestra un dashboard de golden signals. Es el **único servicio de observabilidad expuesto** fuera del host, limitado por la allow list del Security Group.

> **Aclaración:** el borrador original menciona "dos contenedores". Se interpreta como **dos imágenes propias (golden images)**; Prometheus y Grafana se añaden como servicios adicionales con imágenes oficiales de versión fijada. El compose final tiene **4 servicios** (backend, frontend, prometheus, grafana).

## 3. Decisiones

| #   | Decisión | Justificación |
|-----|----------|---------------|
| D1  | Implementar la aplicación fullstack (front + back) descrita en §2.1. | Base representativa para el pipeline. |
| D2  | Backend en Go con capas `domain` / `app` / `ports` / `adapters`. Las dependencias apuntan siempre hacia `domain`. | Testabilidad y adaptadores intercambiables. |
| D3  | Persistencia v1 mediante **adaptador en memoria** detrás de un puerto `ProductRepository`. | Mantiene el compose acotado; la arquitectura hexagonal permite añadir Postgres después sin tocar el dominio. |
| D4  | Métricas con `prometheus/client_golang`, middleware HTTP que registra las 4 golden signals. | Estándar de facto en Go. |
| D5  | Frontend Next.js con **static export** (`output: 'export'`), servido por nginx no-root. Lazy loading con `next/dynamic`. nginx hace **proxy de `/api`** al backend por la red interna, así que el backend **no se publica** en el host y no hace falta CORS. | Comportamiento SPA real, imagen pequeña, sin runtime Node en producción; un solo origen y un puerto menos expuesto. |
| D6  | **Golden images**: multi-stage, base mínima (`distroless`/`alpine`) fijada por digest, usuario no-root, solo el puerto necesario expuesto, sin herramientas de shell innecesarias, `HEALTHCHECK` definido. | Reducir superficie de ataque y hallazgos de Trivy. |
| D7  | **Grafana en contenedor** con datasource Prometheus y dashboard **aprovisionados como código** (`observability/grafana/provisioning/`). Grafana escucha en 3000 dentro del contenedor y se publica como **8081** en el host; el **Security Group** lo abre únicamente a las IPs de la allow list. Prometheus no se expone. La contraseña de admin se pasa por `.env` (no versionado). | Simple: sin dependencias externas ni cuentas de terceros; el acceso se controla en la capa de red. |
| D8  | CI en **GitHub Actions** con tres etapas: **build → sec-check → approve**. | Requisito del ADR. |
| D9  | Escaneo con **Trivy** ejecutado desde su imagen oficial fijada por digest (no se usa `trivy-action`, para reducir dependencias de terceros en el pipeline): `trivy image` (CVEs), `trivy config` (misconfig de Dockerfile, p. ej. usuario root) y `trivy fs --scanners secret`. **Reglas ligeras (proyecto de demostración):** el único bloqueo de misconfig es la política propia **`NP01-PORT-001`** (Rego, HIGH), una *deny list* de puertos inseguros (21 FTP, 22 SSH, 23 Telnet, 135 MS-RPC, 137-139 NetBIOS, 445 SMB, 2375 Docker API, 3389 RDP, 5900 VNC), que también detecta rangos (`EXPOSE 100-200`). Un paso equivalente impide que el compose **publique** esos puertos. Los secretos detectados también bloquean. Las CVEs MEDIUM/HIGH y las buenas prácticas de Dockerfile (p. ej. usuario root) se **reportan sin bloquear**. Resultados SARIF subidos a GitHub Code Scanning. | Requisito explícito: exposición de puertos. Mostrar que el gate funciona sin que CVEs upstream rompan el pipeline de ejemplo. |
| D10 | El **approve** se implementa con un *GitHub Environment* con revisores obligatorios; solo tras la aprobación se publican las imágenes en **GHCR** con tag `sha` + semver. Se publica **la misma imagen escaneada** (se pasa como artefacto entre jobs, sin rebuild). | Gate humano antes de promover artefactos; lo aprobado es exactamente lo publicado. |
| D11 | **Puertos del host**: `8080` → frontend, `8081` → Grafana. Backend (8082) y Prometheus (9090) solo en redes internas del compose. | Alineado con el Security Group: solo hay que abrir 8080 y 8081 (hoy solo están abiertos 22 y 80). |
| D12 | **CD** (`cd.yml`): tras CI exitoso en `main`, despliega por SSH (`ec2-user@EC2_HOST`, secret `AWS_SSH_PRIVATE_KEY`) en `/opt/np01` con `deploy/deploy-ec2.sh`. Ese script fija la release en `release.env`, espera healthchecks y un smoke test, y si fallan vuelve a la release anterior. El `.env` de la VM nunca se sobrescribe. Mismo patrón que el CD de apeiron. | Despliegue simple y reproducible de exactamente lo aprobado y publicado; rollback automático. |

### 3.1 Flujo CI

```
push / PR ──► build (backend, frontend en matriz)
                 │  docker buildx, cache GHA, imagen cargada localmente
                 ▼
             sec-check
                 │  trivy config  → informe MEDIUM/HIGH (no bloquea)
                 │  NP01-PORT-001 → puertos inseguros expuestos ⇒ fail
                 │  compose       → no publica puertos inseguros ⇒ fail
                 │  trivy image   → informe CVEs MEDIUM/HIGH (no bloquea)
                 │  trivy fs      → secretos ⇒ fail
                 │  upload SARIF  → Code Scanning
                 ▼
             approve  (environment: "ci-approval", required reviewers)
                 │
                 ▼
             push a GHCR (solo rama main / tags)
```

## 4. Trade-offs y consecuencias

| Decisión | Ganamos | Cedemos / Riesgo |
|----------|---------|------------------|
| Repositorio en memoria (D3) | Simplicidad, compose pequeño, tests rápidos. | Los datos se pierden al reiniciar; no valida integración con BD real. |
| Next.js static export (D5) | Imagen ligera, sin Node en runtime, menos CVEs. | Se pierden SSR, API routes y middleware de Next. La URL del backend se fija en build (`NEXT_PUBLIC_API_URL`) o se resuelve vía proxy. |
| Distroless (D6) | Superficie de ataque mínima. | Depuración más difícil (sin shell). `HEALTHCHECK` requiere binario propio (p. ej. subcomando `healthcheck` en el binario Go). |
| Trivy ligero: solo bloquean puertos inseguros y secretos (D9) | Pipeline estable para la demo; el gate es fácil de entender y de demostrar. | CVEs y malas prácticas pueden llegar a GHCR si nadie revisa el informe. Para producción se debería volver a bloquear en HIGH/CRITICAL (con `--ignore-unfixed` y `.trivyignore` justificado). |
| Aprobación manual (D10) | Control humano antes de publicar. | Añade latencia al flujo; requiere configurar revisores en el repo. |
| Grafana local + allow list en Security Group (D7) | Sencillo, autocontenido, sin dependencias externas. | Grafana queda expuesto a internet (aunque filtrado por IP): hay que mantener la allow list al día y cambiar la contraseña admin por defecto. Sin TLS, el tráfico va en claro (aceptable para v1; HTTPS con reverse proxy queda para después). Las métricas viven en el host: si se pierde el volumen, se pierde el histórico. |
| CD por SSH (D12) | Simple, sin infraestructura extra; reutiliza el patrón de apeiron. | El puerto 22 debe aceptar a los runners de GitHub; el usuario con docker o sudo equivale a root; `ssh-keyscan` confía en la primera conexión (TOFU). Cada despliegue reinicia el backend y borra los datos en memoria. Alternativa futura: AWS SSM + OIDC. |

## 5. Preguntas abiertas

1. **Modelo maestro–detalle**: se propone `Product` (maestro) → `ProductDetail` (detalle: variantes con SKU, precio, stock). Confirmar si el maestro debería ser `Category` → `Product`.
2. ¿Se requiere persistencia real (Postgres) en v1? Si sí, el compose pasa a 5 servicios.
3. ¿Quiénes son los revisores del environment `ci-approval`?
4. ¿Política de versionado de imágenes (semver manual vs. automática con release-please)?
5. ~~Host y allow list~~ → **Resuelto**: el compose publica 8080 (frontend) y 8081 (Grafana). **Pendiente**: añadir ambas reglas de entrada al Security Group, limitadas a las IPs/CIDR de la allow list (hoy solo están abiertos 22 y 80).

---

## 6. SDD — Spec-Driven Development

Esta sección define las **especificaciones** que guían la implementación. Ninguna tarea se implementa sin su spec y criterios de aceptación; cada PR referencia la spec (`SPEC-xx`) que cumple.

### 6.1 Flujo de trabajo

1. **Specify** — redactar/actualizar la spec en `docs/specs/SPEC-xx-*.md` (qué y por qué, criterios de aceptación).
2. **Plan** — diseño técnico breve dentro de la spec (contratos, estructura, dependencias).
3. **Tasks** — desglosar en tareas pequeñas y verificables (§6.4).
4. **Implement** — código + tests que demuestren cada criterio de aceptación.
5. **Verify** — CI verde y criterios marcados como cumplidos en el PR.

### 6.2 Estructura objetivo del repositorio

```
.
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── domain/          # entidades, reglas, errores de dominio
│   │   ├── app/             # casos de uso
│   │   ├── ports/           # interfaces in/out
│   │   └── adapters/
│   │       ├── http/        # handlers, router, middleware métricas
│   │       ├── memory/      # ProductRepository en memoria
│   │       └── metrics/     # registro Prometheus
│   └── Dockerfile
├── frontend/
│   ├── src/{app,components,hoc,lib}
│   ├── nginx.conf      # SPA + proxy /api → backend
│   └── Dockerfile
├── observability/
│   ├── prometheus/prometheus.yml
│   └── grafana/
│       ├── provisioning/{datasources,dashboards}/
│       └── dashboards/golden-signals.json
├── .github/workflows/ci.yml
├── .trivy/policies/insecure_ports.rego  # NP01-PORT-001 (deny list)
├── .trivyignore
├── .env.example        # GF_SECURITY_ADMIN_PASSWORD (sin valor); .env en .gitignore
├── docker-compose.yml
└── docs/{adrs,specs}/
```

### 6.3 Especificaciones

#### SPEC-01 — Dominio y API de productos (backend)

**Modelo**

| Entidad | Campos |
|---------|--------|
| `Product` (maestro) | `id` (UUID), `name` (req., 1–100), `description`, `createdAt`, `updatedAt`, `details[]` |
| `ProductDetail` (detalle) | `id` (UUID), `productId`, `sku` (req., único), `price` (≥ 0), `stock` (≥ 0) |

**Contrato HTTP** (JSON, prefijo `/api/v1`)

| Método | Ruta | Respuesta OK | Errores |
|--------|------|--------------|---------|
| GET    | `/products` | 200 lista | — |
| GET    | `/products/{id}` | 200 producto con detalles | 404 |
| POST   | `/products` | 201 | 400 validación |
| PUT    | `/products/{id}` | 200 | 400, 404 |
| DELETE | `/products/{id}` | 204 (borra detalles en cascada) | 404 |
| POST   | `/products/{id}/details` | 201 | 400, 404, 409 SKU duplicado |
| PUT    | `/products/{id}/details/{detailId}` | 200 | 400, 404 |
| DELETE | `/products/{id}/details/{detailId}` | 204 | 404 |
| GET    | `/healthz` | 200 | — |
| GET    | `/metrics` | 200 formato Prometheus | — |

Errores con formato uniforme: `{"error": {"code": "...", "message": "..."}}`. CORS habilitado para el origen del frontend.

**Criterios de aceptación**

- [ ] `domain` no importa nada de `adapters`, `net/http` ni librerías de terceros.
- [ ] Los casos de uso dependen solo de interfaces en `ports`.
- [ ] Dado un producto inexistente, cuando hago `GET /products/{id}`, recibo 404 con el formato de error estándar.
- [ ] Dado un SKU ya existente, cuando creo un detalle con ese SKU, recibo 409.
- [ ] Dado `price < 0` o `name` vacío, recibo 400.
- [ ] Cobertura de tests ≥ 80 % en `domain` y `app`.

#### SPEC-02 — Métricas (golden signals)

| Señal | Métrica | Tipo | Labels |
|-------|---------|------|--------|
| Latencia | `http_request_duration_seconds` | Histogram | `method`, `route`, `status` |
| Tráfico | `http_requests_total` | Counter | `method`, `route`, `status` |
| Errores | derivado de `http_requests_total{status=~"5.."}` | — | — |
| Saturación | `http_requests_in_flight` + métricas `go_*`/`process_*` por defecto | Gauge | — |

**Criterios de aceptación**

- [ ] `route` usa el patrón de la ruta (`/products/{id}`), nunca el path real (evita cardinalidad alta).
- [ ] `/metrics` no se instrumenta a sí mismo.
- [ ] Prometheus muestra el target `backend` en estado `UP`.
- [ ] Grafana arranca con el datasource Prometheus (`http://prometheus:9090`) y el dashboard "Golden Signals" (p50/p95/p99 de latencia, RPS, tasa de error, requests en vuelo) ya aprovisionados, sin configuración manual.
- [ ] La contraseña admin de Grafana no es la de por defecto y no está en el repo (`.env` en `.gitignore`; lo verifica `trivy fs --scanners secret`). Registro anónimo deshabilitado.

#### SPEC-03 — Frontend

**Criterios de aceptación**

- [ ] Next.js + TypeScript en modo `strict`, build con `output: 'export'`.
- [ ] Vistas: listado de productos, detalle (maestro + tabla de detalles), formularios de alta/edición para ambos.
- [ ] Al menos un HoC reutilizable (p. ej. `withLoading` / `withErrorBoundary`) aplicado a las vistas que consumen la API.
- [ ] Al menos una vista o componente pesado cargado con `next/dynamic` / `React.lazy` + fallback de carga.
- [ ] Capa `lib/api.ts` única para llamadas HTTP; URL base desde `NEXT_PUBLIC_API_URL` (por defecto `/api/v1`, mismo origen vía proxy nginx).
- [ ] Estilos exclusivamente con Tailwind.
- [ ] Navegación entre vistas sin recarga completa de página.
- [ ] Errores del backend (400/404/409) se muestran al usuario de forma legible.

#### SPEC-04 — Golden images y Docker Compose

**Criterios de aceptación**

- [ ] Dockerfiles multi-stage; imagen base final fijada por **digest**.
- [ ] Backend: binario estático (`CGO_ENABLED=0`) sobre `distroless/static:nonroot`.
- [ ] Frontend: artefactos estáticos servidos por imagen no-root (p. ej. `nginxinc/nginx-unprivileged`).
- [ ] Ambas imágenes: `USER` no-root, un único `EXPOSE` con el puerto de la app, `HEALTHCHECK`, labels OCI (`org.opencontainers.image.*`).
- [ ] Ningún Dockerfile expone puertos de la deny list `NP01-PORT-001`.
- [ ] `docker compose up` levanta los 4 servicios (backend, frontend, prometheus, grafana); solo frontend (**8080**) y Grafana (**8081**) publican puertos al host. Backend, Prometheus y Grafana se comunican por redes internas.
- [ ] El Security Group del host solo abre 8080 y 8081 a las IPs de la allow list; backend (8082) y Prometheus (9090) no son accesibles desde fuera.
- [ ] `depends_on` con `condition: service_healthy` donde aplique.
- [ ] Versiones de Prometheus y Grafana fijadas (sin `latest`).

#### SPEC-05 — Pipeline CI (GitHub Actions)

**Disparadores:** `pull_request` y `push` a `main`/`develop`; tags `v*`.

**Criterios de aceptación**

- [ ] Job `build`: matriz `[backend, frontend]`, `docker/build-push-action` con cache `type=gha`; ejecuta tests unitarios (`go test`, lint/typecheck del front) antes de construir.
- [ ] Job `sec-check` (depende de `build`):
  - [ ] Política `NP01-PORT-001` → falla si un Dockerfile expone un puerto inseguro (22, 137-139, 445, ...).
  - [ ] `trivy config` e `trivy image` → informe MEDIUM/HIGH, **no bloquean**.
  - [ ] `trivy fs --scanners secret` → falla ante cualquier secreto.
  - [ ] Reportes SARIF subidos con `github/codeql-action/upload-sarif`.
- [ ] Job `approve` (depende de `sec-check`): usa `environment: ci-approval` con revisores obligatorios; solo se ejecuta en `main` o tags.
- [ ] Tras aprobación, push a GHCR con tags `sha-<short>` y, en tags, la versión semver.
- [ ] En PRs el pipeline ejecuta `build` + `sec-check` pero **no** publica.
- [ ] Permisos del workflow mínimos (`contents: read`, `packages: write` solo donde se publica, `security-events: write` para SARIF).
- [ ] Actions de terceros fijadas por SHA.
- [ ] Un Dockerfile con `EXPOSE 137-139` (o 22, 445, ...) provoca fallo del pipeline; uno sin `USER` solo aparece en el informe (verificado con un PR de prueba).

### 6.4 Plan de tareas

| Fase | Tarea | Spec | Depende de |
|------|-------|------|------------|
| 0 | **Refinar este ADR** y resolver preguntas abiertas (§5); pasar estado a *Aceptado*. | — | — |
| 0 | Crear `docs/specs/` con SPEC-01…05 a partir de esta sección. | todas | T0 |
| 1 | Scaffold backend (módulo Go, estructura hexagonal, `main`). | 01 | Fase 0 |
| 1 | Dominio + casos de uso + repo en memoria + tests. | 01 | — |
| 1 | Adaptador HTTP + validaciones + errores. | 01 | — |
| 1 | Middleware de métricas + `/metrics`. | 02 | — |
| 2 | Scaffold Next.js + Tailwind + TS strict + static export. | 03 | Fase 0 |
| 2 | Servicio API, HoC, vistas, lazy loading. | 03 | API backend |
| 3 | Golden images (Dockerfiles backend y frontend). | 04 | Fases 1–2 |
| 3 | `docker-compose.yml` + Prometheus (scrape del backend). | 02, 04 | — |
| 4 | Workflow CI: build. | 05 | Fase 3 |
| 4 | Workflow CI: sec-check con Trivy + SARIF. | 05 | — |
| 4 | Environment `ci-approval` + publicación a GHCR. | 05 | — |
| 4 | PR de prueba negativo (Dockerfile inseguro) para validar el gate. | 05 | — |
| 5 | Servicio Grafana en el compose con datasource Prometheus aprovisionado. | 02, 04 | Fase 3 |
| 5 | Dashboard "Golden Signals" aprovisionado como JSON. | 02 | — |
| 5 | Regla de entrada en el Security Group (puerto 3000 solo para la allow list) y verificación de que 9090 no es accesible. | 04 | — |

### 6.5 Definition of Done

- Todos los criterios de aceptación de la spec marcados en el PR.
- Pipeline CI en verde (build + sec-check).
- `docker compose up` funcional desde un clon limpio, documentado en `README.md`.
- Gate de puertos inseguros y de secretos en verde; informe de Trivy revisado.

### 6.6 Tarea actual

> **Ola 1 (27 nov 2026) — implementadas las fases 1 a 5 en una sola entrega.** Verificado en el host: los 4 servicios están `healthy`; CRUD probado de punta a punta a través del proxy; Prometheus tiene el target `backend` en `UP`; Grafana arranca con el datasource y el dashboard ya cargados y rechaza el acceso anónimo; cobertura de tests: `domain` 100 %, `app` 93 %; la política de puertos inseguros bloquea `EXPOSE 22`, `137-139` y `445`, y deja pasar backend y frontend.
>
> **Pendiente:**
> - Abrir 8080 y 8081 en el Security Group (§5.5).
> - Crear el environment `ci-approval` con revisores (§5.3).
> - Ejecutar el pipeline en GitHub. El escaneo de CVEs de imágenes (`trivy image`) no pudo ejecutarse en el host por falta de disco y se validará en el runner.
> - Preguntas 1, 2 y 4 de §5.
