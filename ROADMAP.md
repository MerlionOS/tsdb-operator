# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

### v0.3.1 — 2026-04-13

Three Thanos-sidecar bugs caught during kind verification
(`--storage.tsdb.{min,max}-block-duration`, duplicate `global:` block,
missing `external_labels`). Patch release, recommended for any user on
0.3.0 with `spec.thanos.enabled`.

### v0.3.0 — 2026-04-13

Ecosystem release. Opt-in Thanos sidecar (`spec.thanos.enabled`) plus a
bidirectional `prometheus-operator ↔ tsdb-operator` migration guide.

### v0.2.0 — 2026-04-13

Hardening. REST API wired into the manager + TLS via cert-manager;
end-to-end tests on kind; four real bugs fixed
(double `SetupSignalHandler`, template diff, scheduler never registered,
REST API never started).

### v0.1.0 — 2026-04-13

First tagged release. `PrometheusCluster` CRD, reconciler, HA checker,
S3 backup scheduler, PostgreSQL audit log, gin REST API, Helm chart.

See [`CHANGELOG.md`](CHANGELOG.md) for the per-release details.

## Next up — v0.4.0

Milestone 4 still has two pieces; v0.4.0 picks one (TBD).

- [ ] **`PrometheusClusterSet` CRD.** Cross-namespace aggregation with
  shared backup / audit policy. The flagship multi-cluster feature.
- [ ] **Audit log retention policy.** Partitioned `audit_log` table +
  periodic prune + metrics on row counts. Short, self-contained.

## Later

- [ ] **Smarter backup artifact.** Today the scheduler uploads the
  admin-API JSON response; the on-disk snapshot directory still needs to
  be tarred and shipped for a true point-in-time restore. Tracked at the
  top of [`docs/RESTORE.md`](docs/RESTORE.md).
- [ ] **Webhook validation.** Reject invalid `spec.backup.schedule` cron
  expressions at admission time rather than at cron-fire time.
- [ ] **Per-cluster scrape config.** A user-side way to layer additional
  `scrape_configs` onto the generated `prometheus.yml` without hand-
  editing the ConfigMap.

## Non-goals

To keep scope honest:

- Not reimplementing a TSDB. Prometheus stays the engine.
- Not competing with Thanos / Mimir / VM on global query.
- Not replacing Alertmanager or `vmalert` for alerting.
