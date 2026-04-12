# tsdb-operator vs Thanos vs VictoriaMetrics

中文版: [COMPARISON.zh.md](COMPARISON.zh.md)

A quick orientation for people evaluating TSDB options alongside this operator.

## TL;DR

| | tsdb-operator | Thanos | VictoriaMetrics |
|---|---|---|---|
| **What it is** | Kubernetes operator that manages **vanilla Prometheus** clusters | Horizontally-scalable, long-term storage layer **on top of** Prometheus | Standalone time-series database (single-node + cluster editions) |
| **Primary artifact** | `PrometheusCluster` CRD + controller | Sidecar, Store Gateway, Query, Compactor, Receiver | `vmstorage`, `vminsert`, `vmselect` (or `victoria-metrics` single binary) |
| **Storage model** | Prometheus local TSDB on PVC + scheduled snapshot→S3 | Prometheus local TSDB + object storage for historical blocks | Own TSDB format, local disk (cluster edition shards across `vmstorage`) |
| **Query language** | PromQL (Prometheus) | PromQL (via Thanos Query) | MetricsQL (PromQL-compatible superset) |
| **HA strategy** | Replicate StatefulSet + probe `/-/ready`, delete failed pod | Run ≥2 Prometheus + Thanos Query deduplication | Replicate writes across `vmstorage` nodes (`-replicationFactor`) |
| **Global view** | Not built-in (single cluster at a time) | Yes — Thanos Query fan-out across many Prometheus | Yes — cluster edition + `vmagent` remote_write |
| **Backups** | First-class: cron → S3/MinIO via CRD spec | Implicit (historical blocks live in object storage) | `vmbackup` / `vmrestore` tooling |
| **Audit log** | Built-in (Postgres) | Not provided | Not provided |
| **Management API** | REST (gin): cluster CRUD + backup + audit | No (operated via kubectl / Helm) | No (operated via kubectl / Helm) |
| **License** | Apache 2.0 | Apache 2.0 | Apache 2.0 |

## What tsdb-operator actually is

**A lifecycle controller, not a TSDB.** It still runs upstream Prometheus; the
contribution is declarative HA, scheduled S3 backups, and an audit trail —
the "Day 2" concerns most teams re-invent per project.

## When each fits

### Pick `tsdb-operator` when

- You want to keep **vanilla Prometheus** and just need lifecycle automation
  (provisioning, scaling, HA, off-cluster backups, audit).
- You need a **REST API** over cluster management for a control plane or UI.
- Scale is single-cluster / single-region; global query isn't required.
- Compliance needs an audit log of every operator action.

### Pick Thanos when

- You need a **global query view** over many Prometheus instances.
- You want **unbounded historical retention** in object storage with PromQL.
- You already run Prometheus and want to extend it rather than replace it.
- Downsampling and multi-tenancy matter.

### Pick VictoriaMetrics when

- You want **higher ingest throughput and lower RAM/disk** than Prometheus on
  the same hardware.
- You're willing to run a different TSDB (MetricsQL is PromQL-compatible but
  not identical).
- You want a single vendor stack: `vmagent`, `vmalert`, `vmauth`, `vmbackup`.
- Cluster edition's sharding model fits your scale better than Thanos's
  sidecar/store split.

## They compose

These aren't strictly alternatives:

- `tsdb-operator` could manage the Prometheus instances that **Thanos** queries
  across — the operator handles lifecycle, Thanos handles global view.
- `tsdb-operator`-managed Prometheus can `remote_write` into VictoriaMetrics
  for long-term storage while still giving you the HA/backup/audit surface
  locally.

## Scope boundaries (what this operator does NOT do)

- No global query / federation layer — use Thanos Query or VM cluster.
- No downsampling — Prometheus doesn't do it natively; use Thanos Compactor.
- No multi-tenant label enforcement — use Cortex / Mimir / VictoriaMetrics
  cluster.
- No alerting pipeline — pair with Alertmanager / vmalert.

The goal is a **small, understandable operator** that owns the lifecycle,
not a reimplementation of the wider observability stack.
