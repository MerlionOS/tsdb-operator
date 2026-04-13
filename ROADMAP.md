# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

### v0.10.1 — 2026-04-14

`config-reloader` sidecar replaces the controller-driven reload from
v0.10.0 (which raced kubelet's ConfigMap projection lag). Same pattern
prometheus-operator uses.

### v0.10.0 — 2026-04-14

Auto-reload Prometheus on ConfigMap content change (superseded by
v0.10.1's sidecar approach).

### v0.9.1 — 2026-04-14

Wrap `additional-scrape-configs.yml` under a `scrape_configs:` key so
Prometheus 2.43+ `scrape_config_files` actually accepts it. v0.9.0
wrote a bare list and CrashLooped.

### v0.9.0 — 2026-04-14

`spec.additionalScrapeConfigs` — user-side custom scrape entries
merged into the generated `prometheus.yml` via `scrape_config_files`,
no hand-editing the ConfigMap.

### v0.8.0 — 2026-04-13

`PrometheusClusterSet.spec.backupTemplate` projects onto member CRs
with opt-out annotation; member's own `backup.enabled=true` always wins.

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

## Next up — v1.0 preparation

Stability mode rather than feature mode. The feature set has covered
every Milestone-4 item plus several Later items (auto-overlay, scrape
config layering, sidecar reload). Time to lock down the API surface
and decide what `v1` means.

- [ ] **API stability review.** Walk every field on `PrometheusCluster`,
  `PrometheusClusterSet`, `RemoteWriteSpec`, `S3BackupSpec`,
  `ThanosSpec`, `StorageSpec`. Mark which are `+optional` vs `+required`
  in v1, document semver guarantees.
- [ ] **Breaking change inventory.** Anything that should be renamed
  before v1 freezes the schema (e.g. consider whether
  `additionalScrapeConfigs` should be plural-typed list now to avoid
  later refactor).
- [ ] **Conversion webhook decision.** Will v1 ship as a breaking
  rewrite (`v1` next to `v1alpha1`) requiring conversion, or is the
  current schema close enough to promote in place?
- [ ] **Deprecation policy.** Document how fields will be removed
  post-v1 (one-version warning + alpha annotation).
- [ ] **Plan written up under `docs/V1-PREP.md`.**

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
