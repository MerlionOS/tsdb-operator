# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

### v0.1.0 — 2026-04-13

First tagged release. Operator provisions a Prometheus cluster,
probes replicas for HA, snapshots to S3 on a cron, and exposes a REST
management API with an audit log. Verified end-to-end on kind.

See [`CHANGELOG.md`](CHANGELOG.md) for the full list.

## Next up — v0.2.0

Things we want in the next release. Unchecked items are open work.

- [ ] **TLS on the REST API.** cert-manager integration; terminate TLS on
  the operator service itself, not just an ingress.
- [ ] **Backup flow validated end-to-end on kind+MinIO.** Today the code
  paths are unit-tested but we have not watched a snapshot round-trip
  against a real object store.
- [ ] **Scale / delete / failover smoke tests in the e2e suite** (replaces
  the placeholder e2e).
- [ ] **`tsdb-ctl restore` end-to-end doc + demo.** The CLI works; the
  runbook around it isn't yet written.

## Milestone 4 — Multi-cluster and ecosystem

Larger pieces; individually a 0.x bump each.

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
