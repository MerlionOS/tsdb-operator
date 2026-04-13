# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

### v0.8.0 — 2026-04-13

`PrometheusClusterSet.spec.backupTemplate` now actually projects onto
member CRs. Per-member opt-out via annotation; member's own
`backup.enabled=true` always wins.

### v0.7.0 — 2026-04-13

Validating admission webhook. Invalid `spec.replicas`, missing
`backup.bucket`, bad cron, empty `remoteWrite[].url` are rejected at
`kubectl apply` time. cert-manager-backed TLS via Helm.

### v0.6.0 — 2026-04-13

Real backup artifact. Tar-streams the on-disk snapshot directory out of
the Prometheus pod via SPDY exec → S3 multipart upload, with on-pod
cleanup. Closes the biggest honesty gap in the project.

### v0.5.0 — 2026-04-13

`PrometheusClusterSet` cluster-scoped CRD: cross-namespace aggregation
with per-phase membership status + REST API.

### v0.4.0 — 2026-04-13

Audit-log retention. Logger is instantiated by `cmd/main.go`, `Prune` +
periodic pruner, three new metrics.

### v0.3.1 — 2026-04-13

Three Thanos-sidecar bugs fixed after kind verification.

### v0.3.0 — 2026-04-13

Opt-in Thanos sidecar + bidirectional prometheus-operator migration
guide.

### v0.2.0 — 2026-04-13

Hardening. REST API wired into the manager + cert-manager TLS; four
real bugs fixed from kind verification.

### v0.1.0 — 2026-04-13

First tagged release. Everything core.

See [`CHANGELOG.md`](CHANGELOG.md) for per-release detail.

## Next up — v0.9.0

- [ ] **Per-cluster scrape config layering.** A user-facing
  `spec.additionalScrapeConfigs` (inline YAML or Secret reference) that
  the reconciler merges into the generated `prometheus.yml` without
  users hand-editing the ConfigMap. Today the operator owns the
  ConfigMap and overwrites it on every reconcile, which makes it
  awkward to add custom scrape jobs. This closes that gap.

## Non-goals

To keep scope honest:

- Not reimplementing a TSDB. Prometheus stays the engine.
- Not competing with Thanos / Mimir / VM on global query.
- Not replacing Alertmanager or `vmalert` for alerting.
- **Not building cross-Kubernetes federation.** A
  `PrometheusClusterFederation` that lived in a control cluster and
  reconciled across kubeconfigs was on an earlier roadmap. We won't
  build it, for two reasons: (1) it duplicates mature platforms whose
  whole purpose is multi-cluster delivery — Karmada, Open Cluster
  Management, Argo CD ApplicationSet — any user at that scale already
  runs one of them; (2) doing it honestly is a multi-release scope
  (multi-cluster client pool, cross-cluster watch, unreachable-cluster
  degradation, per-cluster RBAC) and that scope competes with
  vertical-direction work (better scrape config, better backup
  artifact, better audit) that has more direct user value. Recommended
  pattern: propagate `PrometheusCluster` / `PrometheusClusterSet` CRs
  with Argo CD ApplicationSet or Karmada; aggregate Prometheus data
  with Thanos Query.
