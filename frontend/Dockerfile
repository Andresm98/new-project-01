# syntax=docker/dockerfile:1
# Golden image: products SPA (Next.js static export) served by non-root nginx.

ARG NODE_IMAGE=node:26-alpine@sha256:6bd9ab43c0ec0190ca7b2d193906cb8b7edf6e7f4afa43ddeccd6452710bffb5
ARG RUNTIME_IMAGE=nginxinc/nginx-unprivileged:stable-alpine@sha256:f5116ed4606f3eb015dd4a8e28cb9b351476dc0ad4bc6e9f93b19cec130cdf03

FROM ${NODE_IMAGE} AS deps
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

# `docker build --target test .` runs the TypeScript checks (used by CI).
FROM deps AS test
COPY . .
RUN npx next typegen && npm run typecheck

FROM test AS build
ARG NEXT_PUBLIC_API_URL=/api/v1
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
# Optional, local builds only: shown in the footer. Never set it in CI.
ARG NEXT_PUBLIC_GRAFANA_PASSWORD=
ENV NEXT_PUBLIC_GRAFANA_PASSWORD=${NEXT_PUBLIC_GRAFANA_PASSWORD}
RUN npm run build

FROM ${RUNTIME_IMAGE} AS runtime
LABEL org.opencontainers.image.title="np01-frontend" \
      org.opencontainers.image.description="Products master-detail SPA (Next.js + Tailwind)" \
      org.opencontainers.image.source="https://github.com/Andresm98/new-project-01"
COPY --chown=root:root nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build --chown=root:root /app/out /usr/share/nginx/html
USER 101:101
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
