# syntax=docker/dockerfile:1
# Golden image: products API (Go). Multi-stage, distroless, non-root.

ARG GO_IMAGE=golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

FROM ${GO_IMAGE} AS base
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

# `docker build --target test .` runs vet + unit tests (used by CI).
FROM base AS test
RUN gofmt -l . | tee /dev/stderr | (! read) \
 && go vet ./... \
 && go test -cover ./...

FROM base AS build
RUN go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM ${RUNTIME_IMAGE} AS runtime
LABEL org.opencontainers.image.title="np01-backend" \
      org.opencontainers.image.description="Products master-detail REST API with Prometheus metrics" \
      org.opencontainers.image.source="https://github.com/Andresm98/new-project-01"
COPY --from=build /out/api /api
USER 65532:65532
ENV PORT=8082
EXPOSE 8082
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
