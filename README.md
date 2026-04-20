# tsdb-operator

中文: [README.zh.md](README.zh.md)

A Kubernetes operator that manages the full lifecycle of Prometheus clusters:
provisioning, scaling, high availability, scheduled backups to S3-compatible
storage, and an audit log of operator actions.

## What it solves

Running Prometheus at scale means repeatedly wiring up the same primitives —
StatefulSets, PVCs, headless services, health checks, snapshotting, off-cluster
backups, and a record of who changed what. `tsdb-operator` turns that into a
single declarative CRD (`PrometheusCluster`) and a small control plane.

Concretely, pain point → mechanism:

| Pain point | Without the operator | How tsdb-operator handles it |
|------------|---------------------|-----------------------------|
| One cluster = 4 resources to wire up | StatefulSet + PVC + Service + ConfigMap; easy to forget one on change | One `PrometheusCluster` spec; reconciler drives all dependents |
| PVCs don't survive zone/cluster loss | DR is a manual `rsync` | Cron → admin snapshot → S3; `tsdb-ctl restore` pulls it back |
| Replicas don't self-heal | Prometheus has no leader; a stuck pod stays stuck | HA loop probes `/-/ready`, deletes the pod so the StatefulSet rebuilds it |
| "Who changed what, when?" | K8s Events have a TTL; history disappears | Postgres `audit_log` with retention + pruner |
| No cross-namespace view | Clusters scattered across namespaces, listed by hand | `PrometheusClusterSet` (cluster-scoped), selects by label |
| Bad spec only fails at reconcile time | `replicas: 0`, bad cron, empty URL break at runtime | Validating webhook rejects at `kubectl apply` time |
| Custom scrape = hand-edit ConfigMap | One typo and the config reload fails | `spec.additionalScrapeConfigs`, merged by the operator |

Out of scope: long-term storage and cross-cluster query — that's Thanos /
VictoriaMetrics territory. See
[`docs/COMPARISON.en.md`](docs/COMPARISON.en.md).

## Features

**Cluster lifecycle**
- `PrometheusCluster` CRD → StatefulSet + headless Service + ConfigMap + PVC
- Finalizer-based cleanup; phase reporting (`Provisioning` / `Active` / `Scaling` / `Failed`)
- Scale, image upgrade, retention change reconciled via full pod-template diff

**High availability**
- Periodic `/-/ready` probes across replicas
- Unhealthy pods deleted to trigger rescheduling, `LastFailoverTime` + K8s Events recorded

**Backup & restore**
- Cron-driven Prometheus admin snapshot → `tar` streamed via SPDY exec → S3 multipart upload
- On-pod snapshot-dir cleanup after successful upload
- `tsdb-ctl` CLI: `list` / `restore` against any S3-compatible endpoint (MinIO, AWS, etc.)

**Thanos sidecar (opt-in)**
- `spec.thanos.enabled: true` attaches a sidecar sharing the `/prometheus` volume
- Automatic `--enable-feature=expand-external-labels` + per-pod `replica` label
- Object-store config via Secret reference

**Remote write**
- `spec.remoteWrite` renders into `prometheus.yml` with `basicAuth` / `bearerToken` Secret refs

**Multi-namespace aggregation**
- `PrometheusClusterSet` (cluster-scoped CRD) selects clusters by label across namespaces
- Status reports member count, per-phase histogram, and member list

**Admission validation (opt-in)**
- Validating webhook rejects bad `spec.replicas`, missing `backup.bucket`, bad cron,
  empty `remoteWrite[].url` at `kubectl apply` time
- cert-manager-backed TLS via Helm values

**Audit log (opt-in)**
- PostgreSQL-backed `audit_log`, every cluster mutation + backup event recorded
- Retention policy (`--audit-retention-days`) with periodic pruner

**REST API (opt-in)**
- gin-based: `/api/clusters`, `/api/clustersets`, `/api/clusters/:name/{backup,audit}`
- cert-manager TLS supported

**Observability**
- Prometheus metrics: `tsdb_operator_{cluster_phase,backup_total,failover_total,audit_*}`
- Grafana dashboard at `grafana/dashboards/tsdb-operator.json`

**Packaging**
- Helm chart at `charts/tsdb-operator/` with feature flags for every subsystem
- envtest + kind e2e; every release verified against a real kind cluster

> Why snapshot to S3 when PVCs exist? See
> [`docs/BACKUPS.en.md`](docs/BACKUPS.en.md) ([中文](docs/BACKUPS.zh.md)).
>
> How to restore from a backup:
> [`docs/RESTORE.md`](docs/RESTORE.md) ([中文](docs/RESTORE.zh.md)).
>
> Migrating to/from prometheus-operator:
> [`docs/MIGRATION.md`](docs/MIGRATION.md) ([中文](docs/MIGRATION.zh.md)).
>
> Cluster-scoped aggregation across namespaces:
> [`docs/CLUSTERSET.md`](docs/CLUSTERSET.md) ([中文](docs/CLUSTERSET.zh.md)).
>
> How does this compare to Thanos and VictoriaMetrics? See
> [`docs/COMPARISON.en.md`](docs/COMPARISON.en.md) ([中文](docs/COMPARISON.zh.md)).
>
> Broader TSDB landscape (Prometheus ecosystem + general-purpose TSDBs):
> [`docs/TSDB-LANDSCAPE.en.md`](docs/TSDB-LANDSCAPE.en.md)
> ([中文](docs/TSDB-LANDSCAPE.zh.md)).
>
> Mainland China observability landscape (Nightingale / DeepFlow / ARMS /
> domestic TSDBs): [`docs/CHINA-LANDSCAPE.en.md`](docs/CHINA-LANDSCAPE.en.md)
> ([中文](docs/CHINA-LANDSCAPE.zh.md)).
>
> Prometheus TSDB internals that informed this operator's design:
> [`docs/TSDB-INTERNALS.en.md`](docs/TSDB-INTERNALS.en.md)
> ([中文](docs/TSDB-INTERNALS.zh.md)).
>
> Adding custom scrape configs without hand-editing the ConfigMap:
> [`docs/SCRAPE-CONFIGS.md`](docs/SCRAPE-CONFIGS.md)
> ([中文](docs/SCRAPE-CONFIGS.zh.md)).
>
> Plan for the v1.0 API freeze:
> [`docs/V1-PREP.md`](docs/V1-PREP.md)
> ([中文](docs/V1-PREP.zh.md)).

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
# 1. Install the operator via Helm
helm install tsdb-operator ./charts/tsdb-operator -n tsdb-operator --create-namespace

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

## Roadmap

See [`ROADMAP.md`](ROADMAP.md) ([中文](ROADMAP.zh.md)).

## Architecture Decision Records

See [`docs/adr/`](docs/adr/) for the rationale behind key choices.

## The Observability Epic (trilogy)

A long-form essay tracing observability from ancient beacon towers to
modern eBPF — and why Chinese and Western observability ecosystems
diverged along lines already written three thousand years ago.

- **Book I · [The Flame and the Eyes](drafts/wechat-epic-1-flame-and-eyes.en.md)** ([中文](drafts/wechat-epic-1-flame-and-eyes.zh.md)) — 771 BCE → 1858 CE: Eastern Qintianjian vs. Western Nightingale
- **Book II · [Iron and Lightning](drafts/wechat-epic-2-iron-and-lightning.en.md)** ([中文](drafts/wechat-epic-2-iron-and-lightning.zh.md)) — 1837 → 2019: from Morse to OpenTelemetry
- **Book III · [The Divergence and the Echo](drafts/wechat-epic-3-divergence-and-echo.en.md)** ([中文](drafts/wechat-epic-3-divergence-and-echo.zh.md)) — 2013 → today: the modern Sino-American divergence, and where `tsdb-operator` sits

## License

Apache 2.0
