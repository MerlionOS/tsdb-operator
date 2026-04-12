# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Project

`tsdb-operator` is a Kubernetes operator that manages the lifecycle of
Prometheus clusters as a single `PrometheusCluster` CRD: provisioning,
scaling, HA health checking, scheduled S3/MinIO backups, and a PostgreSQL
audit log. A gin REST API exposes the same operations over HTTP.

- Module: `github.com/MerlionOS/tsdb-operator`
- API group / version: `observability.merlionos.org/v1`
- Kind: `PrometheusCluster`

## Layout

```
api/v1/                  CRD types (edit → run `make generate manifests`)
internal/controller/     PrometheusCluster reconciler (StatefulSet + SVC)
internal/ha/             Health-check loop + failover
internal/backup/         Cron snapshot scheduler + S3 uploader
internal/audit/          PostgreSQL audit log (sqlx + lib/pq)
pkg/api/                 gin HTTP server
config/                  kubebuilder kustomize manifests
grafana/dashboards/      Operator dashboard
```

## Tech stack

- Go 1.22+, controller-runtime, kubebuilder v3
- gin (REST API), sqlx + lib/pq (audit), aws-sdk-go-v2 (S3), robfig/cron
- envtest for integration tests

## Conventions

- Structured logging via `logf.FromContext(ctx)` (controller-runtime/zap). No `fmt.Println`.
- Wrap errors with `fmt.Errorf("...: %w", err)`.
- Always propagate `context.Context`.
- After changing anything in `api/v1/`, run `make generate manifests`.
- RBAC markers live on the reconciler; regenerate with `make manifests`.
- Every controller/package should have unit tests; reconciler uses envtest.

## Common commands

```bash
make generate manifests   # regenerate deepcopy + CRDs after type changes
make test                 # unit + envtest
make test-e2e             # kind-based e2e
make build                # build manager binary
make docker-build IMG=... # build image
make install              # install CRDs into current kube context
make dev                  # kind cluster + install + deploy
docker compose up -d      # local Postgres + MinIO + Grafana
```

## Things to avoid

- Don't commit generated binaries under `bin/`.
- Don't hand-edit `zz_generated.deepcopy.go` or `config/crd/bases/*.yaml` —
  regenerate instead.
- Don't introduce `fmt.Println` / `log.Printf`; use the contextual logger.
