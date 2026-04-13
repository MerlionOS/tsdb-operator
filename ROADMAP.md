# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

### v0.5.0 — 2026-04-13

Multi-cluster aggregation. `PrometheusClusterSet` cluster-scoped CRD
groups `PrometheusCluster`s by label across namespaces; status reports
membership + per-phase counts. REST API + envtest + kind-verified.

### v0.4.0 — 2026-04-13

Audit-log hardening. The logger is finally instantiated by
`cmd/main.go`; adds `Prune` + periodic pruner, three new metrics, and
Helm chart plumbing.

### v0.3.1 — 2026-04-13

Three Thanos-sidecar bugs fixed after kind verification.

### v0.3.0 — 2026-04-13

Opt-in Thanos sidecar + bidirectional prometheus-operator migration
guide.

### v0.2.0 — 2026-04-13

Hardening. REST API wired into the manager + cert-manager TLS; four real
bugs fixed from kind verification.

### v0.1.0 — 2026-04-13

First tagged release. Everything core.

See [`CHANGELOG.md`](CHANGELOG.md) for per-release detail.

## Milestone 4 — done ✅

All four Milestone-4 items shipped: Thanos sidecar (v0.3), migration
guide (v0.3), audit retention (v0.4), `PrometheusClusterSet` (v0.5).

## Later

Open follow-ups. Not yet grouped into a release.

- [ ] **Auto-overlay `backupTemplate` onto Set members.** v0.5.0
  records the template in spec but doesn't mutate member CRs. Needs an
  "owner of truth" policy (always overlay vs only-if-empty) and a way
  for members to opt out.
- [ ] **Smarter backup artifact.** Today the scheduler uploads the
  admin-API JSON; the on-disk snapshot directory still needs to be
  tarred and shipped for a true point-in-time restore. Tracked at the
  top of [`docs/RESTORE.md`](docs/RESTORE.md).
- [ ] **Admission webhook.** Reject invalid `spec.backup.schedule`
  cron expressions at admission time rather than at cron-fire time.
- [ ] **Per-cluster scrape config.** A user-side way to layer additional
  `scrape_configs` onto the generated `prometheus.yml` without hand-
  editing the ConfigMap.
- [ ] **Cross-Kubernetes federation.** Today a `PrometheusClusterSet`
  spans namespaces, not clusters. A future `PrometheusClusterFederation`
  could aggregate across kubeconfigs.

## Non-goals

- Not reimplementing a TSDB. Prometheus stays the engine.
- Not competing with Thanos / Mimir / VM on global query.
- Not replacing Alertmanager or `vmalert` for alerting.
