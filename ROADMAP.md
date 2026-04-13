# Roadmap

中文版: [ROADMAP.zh.md](ROADMAP.zh.md)

What's shipped, what's next, what we're deliberately not doing.

## Shipped

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

## Next up — v0.7.0

- [ ] **Admission webhook.** Reject invalid `spec.backup.schedule` cron
  expressions and other bad spec shapes at admission time rather than at
  cron-fire time. Validating webhook + cert-manager plumbing in the
  chart.

## Later

- [ ] **Auto-overlay `backupTemplate` onto Set members.** v0.5.0 records
  the template in spec but doesn't mutate member CRs. Needs an "owner of
  truth" policy (always vs only-if-empty) and a way for members to opt
  out.
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
