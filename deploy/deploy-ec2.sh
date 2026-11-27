#!/usr/bin/env bash
# Deploys a np01 release on the EC2 host. Run as root (CD calls it via sudo).
#
#   IMAGE_PREFIX  ghcr.io/<owner>/<repo>          (required)
#   IMAGE_TAG     sha-<commit>                     (required)
#   DEPLOY_DIR    default /opt/np01
#   BUNDLE        default /tmp/np01-bundle.tgz    (compose + observability + .env.example)
#   PULL          default 1; 0 only for local tests with images already present
#
# The VM owns .env: it is created once (random Grafana password) and never
# overwritten. If the new release is not healthy, the previous one is restored.
set -euo pipefail

: "${IMAGE_PREFIX:?IMAGE_PREFIX is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/np01}"
BUNDLE="${BUNDLE:-/tmp/np01-bundle.tgz}"
PULL="${PULL:-1}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"
APP_URL="${APP_URL:-http://127.0.0.1:8080}"

log() { echo "[deploy] $*"; }

install -d -m 755 "$DEPLOY_DIR"
tar -xzf "$BUNDLE" -C "$DEPLOY_DIR" --no-same-owner
cd "$DEPLOY_DIR"

if [ ! -f .env ]; then
  log "creating .env from .env.example with a random Grafana password"
  password=$(head -c 24 /dev/urandom | base64 | tr -d '/+=')
  (umask 077 && sed "s/^GRAFANA_ADMIN_PASSWORD=.*/GRAFANA_ADMIN_PASSWORD=${password}/" .env.example > .env)
fi

# release.env pins the images; release.env.prev is the rollback target.
[ -f release.env ] && cp release.env release.env.prev
cat > release.env <<EOF
BACKEND_IMAGE=${IMAGE_PREFIX}/backend:${IMAGE_TAG}
FRONTEND_IMAGE=${IMAGE_PREFIX}/frontend:${IMAGE_TAG}
EOF

compose() { docker compose --env-file .env --env-file release.env "$@"; }

wait_healthy() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT)) expected ids id status pending
  expected=$(compose config --services | wc -l)
  while :; do
    ids=$(compose ps -a -q)
    pending=0
    for id in $ids; do
      status=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$id")
      [ "$status" = "healthy" ] || pending=1
    done
    if [ "$pending" -eq 0 ] && [ "$(wc -w <<<"$ids")" -eq "$expected" ]; then
      return 0
    fi
    if [ "$SECONDS" -ge "$deadline" ]; then
      log "services not healthy after ${HEALTH_TIMEOUT}s"
      compose ps -a
      return 1
    fi
    sleep 5
  done
}

smoke_test() {
  curl -fsS --max-time 5 "$APP_URL/healthz" >/dev/null &&
    curl -fsS --max-time 5 "$APP_URL/api/v1/products" >/dev/null
}

# release [pull]: pull=0 reuses local images (rollback: the previous release
# is always kept locally, and may not exist in the registry).
release() {
  local pull="${1:-$PULL}"
  log "releasing: $(tr '\n' ' ' < release.env)"
  if [ "$pull" = "1" ]; then
    compose pull --quiet || return 1
  fi
  compose up -d --no-build --remove-orphans || return 1
  wait_healthy || return 1
  smoke_test || { log "smoke test failed"; return 1; }
}

if release; then
  log "release ${IMAGE_TAG} is healthy"
else
  log "release ${IMAGE_TAG} failed"
  if [ -f release.env.prev ]; then
    log "rolling back"
    mv release.env.prev release.env
    if release 0; then
      log "rollback OK"
    else
      log "rollback FAILED: manual action required"
    fi
  fi
  exit 1
fi

# Remove old np01 release images; keep the current and the previous one.
# Only this project's GHCR images are touched (other stacks on the VM are safe).
keep=$(cut -d= -f2 release.env release.env.prev 2>/dev/null || true)
docker images --format '{{.Repository}}:{{.Tag}}' |
  grep -E "^${IMAGE_PREFIX}/(backend|frontend):" |
  while read -r image; do
    grep -qxF "$image" <<<"$keep" || docker rmi "$image" >/dev/null 2>&1 || true
  done

compose ps
