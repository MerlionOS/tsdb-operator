# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's planned next, grouped by intent. Order inside each group is the
suggested build order, not a hard schedule.

## Milestone 1 — Make it actually run

The current scaffold compiles and passes CI but would not survive a real
`kubectl apply`. These are the concrete defects to close first.

- [ ] **Mount a Prometheus config.** The StatefulSet points at
  `/etc/prometheus/prometheus.yml` but nothing is mounted there → CrashLoop.
  Generate a default ConfigMap per `PrometheusCluster`, mount it, and let
  users override via `spec.configMapRef`.
- [ ] **Enable the admin API for snapshots.** Add `--web.enable-admin-api`
  to the container args when `spec.backup.enabled` is true. Without this,
  the snapshot endpoint returns 404 and backups silently fail.
- [ ] **Wire HA and Backup controllers into the manager.** Today only the
  `PrometheusCluster` reconciler is registered in `cmd/main.go`;
  `internal/ha` and `internal/backup` are built but never started.
  Register them via `mgr.Add(...)` behind flags (`--enable-ha`,
  `--enable-backup`).
- [ ] **Finalizer on `PrometheusCluster`.** Ensure the headless Service and
  (optionally) the last backup artifact are cleaned up on delete. Without
  this, orphaned resources accumulate.

## Milestone 2 — Observable and testable

- [ ] **Prometheus metrics.** Register the metrics the Grafana dashboard
  already references:
  - `tsdb_operator_cluster_phase{cluster,phase}`
  - `tsdb_operator_backup_total{cluster,result}`
  - `tsdb_operator_failover_total{cluster}`
- [ ] **Envtest suite for the reconciler.** Cover create / scale / delete
  and the phase transitions.
- [ ] **Unit tests for HA and Backup.** Use a fake HTTP server + fake S3
  `Uploader`; assert `LastFailoverTime` / `LastBackupTime` are updated.
- [ ] **REST API contract tests.** Spin up the gin router with a fake
  client, exercise every route.

## Milestone 3 — Day-2 polish

- [ ] **`tsdb-ctl restore` CLI.** Pull a snapshot from S3 back into a PVC
  (the symmetric half of backup).
- [ ] **Helm chart.** `charts/tsdb-operator/` with operator install +
  values for Postgres/S3 secrets.
- [ ] **Kustomize `remote_write` integration.** Optional `spec.remoteWrite`
  so a managed Prometheus can push to Thanos / Mimir / VictoriaMetrics.
- [ ] **TLS on the REST API.** cert-manager integration; terminate TLS on
  the operator service itself, not just an ingress.
- [ ] **ADRs under `docs/adr/`.** Record why this operator exists separately
  from prometheus-operator, why snapshots over continuous remote-write,
  etc.

## Milestone 4 — Multi-cluster / ecosystem

Aspirational; only worth doing once Milestones 1–3 are solid.

- [ ] **Cluster federation CRD.** `PrometheusClusterSet` spanning multiple
  namespaces with shared backup/audit.
- [ ] **Thanos sidecar opt-in.** `spec.thanos.enabled: true` attaches a
  sidecar and an objstore config secret.
- [ ] **Audit log retention policy.** Partitioned table + periodic prune.
- [ ] **Operator-to-operator migration guide** from prometheus-operator to
  tsdb-operator (and the inverse).

## Non-goals

To keep scope honest:

- Not reimplementing a TSDB. Prometheus stays the engine.
- Not competing with Thanos / Mimir / VM on global query.
- Not replacing Alertmanager or `vmalert` for alerting.
