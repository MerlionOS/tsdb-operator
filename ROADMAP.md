# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

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

## Next up — v0.8.0

- [ ] **Auto-overlay `backupTemplate` onto Set members.** v0.5.0 records
  the template in spec but doesn't mutate member CRs. This closes that
  loop so `PrometheusClusterSet` becomes a real policy object, not just
  a dashboard.
  - Policy: overlay only when the member's `spec.backup.enabled` is
    unset / false; members always win on any field they explicitly set.
  - Opt-out: per-member annotation
    `observability.merlionos.org/clusterset-opt-out: "true"`.
  - Scope: one new Set→member projection pass in the Set reconciler,
    conflict detection, owner-reference decision (don't re-parent,
    just label), envtest coverage.

## Later

Smaller scope next to v0.8.0; not yet committed to a release.

- [ ] **Per-cluster scrape config layering.** `spec.additionalScrapeConfigs`
  (inline YAML or secret ref) merged into the generated `prometheus.yml`
  without users hand-editing the ConfigMap. Highest user value among the
  remaining items; medium scope.
- [ ] **Cross-Kubernetes federation.** A future
  `PrometheusClusterFederation` aggregates `PrometheusClusterSet`s
  across kubeconfigs. Largest item here: needs multi-cluster client
  management, auth, cross-cluster watch. Likely two releases, not one.

## Non-goals

- Not reimplementing a TSDB. Prometheus stays the engine.
- Not competing with Thanos / Mimir / VM on global query.
- Not replacing Alertmanager or `vmalert` for alerting.
