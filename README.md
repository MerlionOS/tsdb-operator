# tsdb-operator

A Kubernetes operator that manages the full lifecycle of Prometheus clusters:
provisioning, scaling, high availability, scheduled backups to S3-compatible
storage, and an audit log of operator actions.

## What it solves

Running Prometheus at scale means repeatedly wiring up the same primitives —
StatefulSets, PVCs, headless services, health checks, snapshotting, off-cluster
backups, and a record of who changed what. `tsdb-operator` turns that into a
single declarative CRD (`PrometheusCluster`) and a small control plane.

> How does this compare to Thanos and VictoriaMetrics? See
> [`docs/COMPARISON.md`](docs/COMPARISON.md).

## Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                         tsdb-operator                         │
│                                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ PrometheusCluster      │   │ HA health checker         │   │
│  │ reconciler             │──▶│ (probe /-/ready, failover)│   │
│  │ (StatefulSet + SVC)    │   └───────────────────────────┘   │
│  └────────────┬───────────┘                                   │
│               │                                               │
│               ▼                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ Backup scheduler       │──▶│ S3 / MinIO                │   │
│  │ (cron, admin snapshot) │   └───────────────────────────┘   │
│  └────────────────────────┘                                   │
│                                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ REST API (gin)         │──▶│ Audit log (PostgreSQL)    │   │
│  └────────────────────────┘   └───────────────────────────┘   │
└───────────────────────────────────────────────────────────────┘
```

## Quick start

```bash
# 1. Spin up a local cluster and install CRDs + operator
make dev

# 2. Create a PrometheusCluster
kubectl apply -f config/samples/observability_v1_prometheuscluster.yaml

# 3. Watch it come up
kubectl get prometheuscluster -w
```

## CRD example

See [`config/samples/observability_v1_prometheuscluster.yaml`](config/samples/observability_v1_prometheuscluster.yaml).

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: demo
spec:
  replicas: 2
  retention: 15d
  storage:
    size: 20Gi
  backup:
    enabled: true
    bucket: tsdb-backups
    schedule: "0 */6 * * *"
```

## REST API

| Method | Path                              | Description                       |
|--------|-----------------------------------|-----------------------------------|
| GET    | `/api/clusters`                   | List PrometheusCluster resources  |
| POST   | `/api/clusters`                   | Create a cluster                  |
| GET    | `/api/clusters/:name`             | Get cluster + status              |
| DELETE | `/api/clusters/:name`             | Delete cluster                    |
| POST   | `/api/clusters/:name/backup`      | Trigger manual backup             |
| GET    | `/api/clusters/:name/audit`       | Query audit log                   |

Set `X-Operator: <user>` on write requests to record the actor in the audit log.

## Development

```bash
# Local dependencies (Postgres for audit, MinIO for backup, Grafana)
docker compose up -d

# Regenerate CRDs / deepcopy after changing api/v1 types
make generate manifests

# Unit + envtest
make test

# End-to-end on kind
make test-e2e
```

### Layout

```
api/v1/                        CRD types
internal/controller/           PrometheusCluster reconciler
internal/ha/                   Health checking + failover
internal/backup/               Snapshot + S3 upload scheduler
internal/audit/                PostgreSQL audit log
pkg/api/                       gin HTTP server
config/                        kustomize manifests (kubebuilder)
grafana/dashboards/            Operator dashboard JSON
```

## License

Apache 2.0
