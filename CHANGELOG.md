# Changelog

All notable changes to this project are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.9.0] — 2026-04-14

User-side custom scrape config without hand-editing the operator's
ConfigMap.

### Added

- `spec.additionalScrapeConfigs` (string, top-level YAML list) on
  `PrometheusCluster`. Stored under ConfigMap key
  `additional-scrape-configs.yml`; main `prometheus.yml` references it
  via Prometheus 2.43+ `scrape_config_files` directive.
- Webhook validation: rejects non-list YAML at `kubectl apply` time
  with the field path; doesn't try to validate scrape-config internals
  (Prometheus reload still surfaces those).
- `docs/SCRAPE-CONFIGS.{en,zh}.md` with example and explicit limits
  (inline-only, no auto-reload, no PodMonitor/ServiceMonitor).
- Two new tests in `internal/controller/render_test.go` and two in
  `internal/webhook/`.

[0.9.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.9.0

## [0.8.0] — 2026-04-13

`PrometheusClusterSet.spec.backupTemplate` now actually takes effect on
member CRs. Closes the Deferred item from v0.5.0.

### Added

- `PrometheusClusterSetReconciler.overlayBackup`: copies `spec.backupTemplate`
  onto each matched member whose own `spec.backup.enabled` is false and
  that does not carry the opt-out annotation. Stamps
  `observability.merlionos.org/clusterset: <set-name>` for traceability.
- `OptOutAnnotation` constant
  (`observability.merlionos.org/clusterset-opt-out`): members with value
  `"true"` are never touched by any Set.
- Three new envtest specs covering overlay / opt-out / member-wins.
- `docs/CLUSTERSET.{en,zh}.md` grew an **Overlay rules** section plus
  the non-obvious edge cases (deletion doesn't unwind, ownership
  transfer back to the user).

### Policy (all-or-nothing per member)

- Member wins when `spec.backup.enabled` is already true.
- Opt-out annotation always wins.
- Otherwise the member's `spec.backup` is replaced wholesale with the
  template plus `enabled: true`. No field-level merge.

[0.8.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.8.0

## [0.7.0] — 2026-04-13

Admission-time validation. Invalid specs are now rejected at
`kubectl apply` time instead of crashing the reconciler or silently
never firing a cron. Opt-in behind `features.webhook=true`.

### Added

- `internal/webhook.PrometheusClusterValidator` — validating admission
  webhook (controller-runtime typed `Validator[T]`). Rejects:
  - `spec.replicas < 1`
  - `spec.backup.enabled=true` with empty `spec.backup.bucket`
  - `spec.backup.schedule` that fails `cron.ParseStandard`
  - `spec.remoteWrite[].url` empty
- `cmd/main.go` flag `--enable-webhook`; uses the existing
  `--webhook-cert-path` plumbing.
- Helm chart: `features.webhook`, `webhook.*` values. When enabled,
  the chart creates a cert-manager `Issuer`+`Certificate` (self-signed
  default) + Service + `ValidatingWebhookConfiguration` with the
  `cert-manager.io/inject-ca-from` annotation.
- Unit tests covering each rejection path + the happy path.
- Verified on kind: `kubectl apply` of invalid specs gets the webhook's
  specific error message.

[0.7.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.7.0

## [0.6.0] — 2026-04-13

Real backups. Closes the biggest honesty gap in the project: from v0.1.0
until now the scheduler uploaded the admin-API JSON response as the
"backup artifact" — a marker, not something you could restore from. This
release replaces that with a proper tar of the on-disk snapshot
directory, streamed via S3 multipart, with on-pod cleanup afterwards.
Verified end-to-end on kind+MinIO: a 1-minute cron produced 108–112 KiB
tarballs containing real TSDB blocks (chunks, index, meta.json).

### Added

- `PodExecutor` interface and `SPDYExecutor` implementation using
  `k8s.io/client-go/tools/remotecommand`. Invoked with
  `tar -C /prometheus/snapshots -cf - <snapshot-name>` to stream the
  snapshot dir out of the pod.
- Multipart streaming upload via the s3 `manager.Uploader`
  (`backup.S3Client.StreamUpload`). Required because `PutObject` rejects
  unseekable pipe readers over plain HTTP.
- Snapshot admin-API response parser — the returned directory name is
  what gets tarred and then deleted.
- Best-effort cleanup: `rm -rf /prometheus/snapshots/<name>` after a
  successful upload so snapshot dirs don't accumulate on the PVC.
- RBAC: `pods/exec` create verb added to the Helm ClusterRole.
- Fallback path preserved: when `Exec` is nil (unit tests, non-cluster
  contexts), the scheduler still uploads the admin-API response so
  existing tests keep their shape.

### Changed

- `backup.Uploader` interface gains `StreamUpload`. Existing PutObject
  callers continue to work; streaming goes through the new method.
- `docs/RESTORE.{en,zh}.md` header rewritten — no more "this is a marker,
  adjust the tar step yourself" caveat.

### Fixed

- `PutObject, compute input header checksum failed, unseekable stream is
  not supported without TLS and trailing checksum` — surfaced during
  kind verification when piping an `io.PipeReader` into PutObject.

[0.6.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.6.0

## [0.5.0] — 2026-04-13

Multi-cluster aggregation. Adds the `PrometheusClusterSet` cluster-scoped
CRD, the flagship Milestone-4 feature: groups `PrometheusCluster`
resources by label across namespaces and reports membership + per-phase
counts in the Set's status.

### Added

- `PrometheusClusterSet` (cluster-scoped) with `spec.clusterSelector`,
  `spec.namespaceSelector`, and `spec.backupTemplate`.
- Set reconciler that watches `PrometheusCluster` events and refreshes
  every Set's `status.{memberCount,phaseCount,members}`.
- REST API: `GET /api/clustersets`, `GET /api/clustersets/:name`.
- RBAC: cluster-scoped read on `prometheusclustersets` and `namespaces`,
  plus the new status/finalizers verbs in the Helm chart's ClusterRole.
- Envtest specs covering label-match and "match everything" branches.
- `docs/CLUSTERSET.{en,zh}.md` describing the model, REST surface, and
  what is deliberately out of scope (no auto-overlay of the
  `backupTemplate`, no cross-Kubernetes federation).

### Deferred

- Mutating member CRs from the Set's `backupTemplate`. Tracked under
  "Later" in the roadmap.

[0.5.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.5.0

## [0.4.0] — 2026-04-13

Audit-log hardening. The `audit.Logger` has existed since v0.1.0 but was
never actually instantiated by `cmd/main.go` — a gap this release closes,
alongside a retention policy and metrics so the `audit_log` table
doesn't grow forever.

### Added

- `cmd/main.go` flags: `--audit-dsn` (or `TSDB_AUDIT_DSN` env),
  `--audit-retention-days`. When DSN is set, the logger is opened on
  startup and handed to the REST API server, so
  `GET /api/clusters/:name/audit` and the auto-recorded entries on
  create/delete/backup routes actually work.
- `audit.Logger.Prune(ctx, before)` deletes rows older than the given
  cutoff and returns the count.
- `audit.Logger.Start(ctx)` is a `manager.Runnable` pruner. Runs every
  hour (configurable via `PruneInterval`) plus once on startup so the
  effect is visible immediately. No-op when `RetentionDays == 0`.
- Metrics: `tsdb_operator_audit_record_total{cluster,result}`,
  `tsdb_operator_audit_prune_total`, `tsdb_operator_audit_rows`.
- Helm chart: `features.audit`, `audit.dsnSecretRef`,
  `audit.retentionDays`. DSN is plumbed via `TSDB_AUDIT_DSN` env from
  a Secret.
- Tests for `Prune`, `RowCount`, and pruner-start no-op using
  `DATA-DOG/go-sqlmock`.

[0.4.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.4.0

## [0.3.1] — 2026-04-13

Patch release. Three real bugs in the v0.3.0 Thanos sidecar path were
caught during kind verification and fixed. Upgrade from 0.3.0 is
recommended for anyone who enabled `spec.thanos.enabled`.

### Fixed

- Prometheus container now gets `--storage.tsdb.min-block-duration=2h`
  and `--storage.tsdb.max-block-duration=2h` when Thanos is enabled.
  Without these flags the sidecar refuses to start with "Compaction
  needs to be disabled".
- `renderConfig` previously emitted two `global:` blocks when Thanos was
  enabled, causing `parsing YAML file /etc/prometheus/prometheus.yml:
  field global already set`. It now composes one combined `global:`
  section.
- Thanos requires uniquely-identifying external labels per replica.
  `renderConfig` now injects `cluster: <name>` and `replica:
  ${POD_NAME}`, Prometheus gets `--enable-feature=expand-external-labels`,
  and `POD_NAME` is wired via the downward API so the replica token
  resolves per pod.

## [0.3.0] — 2026-04-13

Ecosystem release. Adds an opt-in Thanos sidecar so tsdb-operator-managed
Prometheus instances can participate in a global query view and long-term
object-storage retention without swapping operators. Ships with a
bidirectional migration guide.

### Added

- `spec.thanos.enabled` attaches a Thanos sidecar to each replica. The
  sidecar shares the `/prometheus` data volume and ships 2h blocks to
  object storage. Configurable via `spec.thanos.image` and
  `spec.thanos.objectStorageConfigSecretRef` (references a Secret with an
  `objstore.yml` key, mounted via `--objstore.config-file`).
- `docs/MIGRATION.{en,zh}.md`: bidirectional prometheus-operator ↔
  tsdb-operator guide with field-by-field CR translation tables and a
  "run both" option.
- Unit tests covering the sidecar-off, sidecar-on-no-objstore, and
  sidecar-on-with-objstore branches.

## [0.2.0] — 2026-04-13

Hardening release. Every v0.1.0 feature that existed in the codebase but
was never actually executed end-to-end has now been run against a real
kind cluster — with backups going to MinIO and the REST API served over
cert-manager-issued TLS. Several real bugs were found and fixed as a
direct result of this exercise.

### Added

- REST API wired into the manager process (`--enable-api`,
  `--api-address`, `--api-namespace`, `--api-tls-cert-dir`). It was shipped
  as a package in 0.1.0 but never started by `cmd/main.go`.
- TLS support for the REST API via a mounted cert directory, plus Helm
  chart plumbing for cert-manager: `Issuer` (self-signed default) and
  `Certificate` resources with in-cluster DNS SANs.
- Helm chart: `api` Service, `api.tls.enabled` / `api.tls.certManager.*`
  values, and `s3.credentialsSecretName` for wiring backup credentials.
- End-to-end test suite for `PrometheusCluster` lifecycle (create, scale,
  backup-toggle, finalizer) in `test/e2e/`, plus an `E2E_SKIP_SETUP=true`
  escape hatch for running specs against an already-deployed operator.
- Restore runbook at `docs/RESTORE.{en,zh}.md`, verified against MinIO.

### Fixed

- `cmd/main.go` called `ctrl.SetupSignalHandler()` twice, causing
  `panic: close of closed channel` at startup. The handler is now set up
  once and shared between the AWS config load and `mgr.Start`.
- Reconciler only updated the `StatefulSet` template when `spec.replicas`
  changed, so toggling `spec.backup.enabled` did not flip the container
  args. It now compares the full pod template via
  `equality.Semantic.DeepEqual` and patches on any drift.
- Backup `Scheduler.Start` was registered as a manager runnable but no
  cluster was ever registered with it, so cron never fired. The scheduler
  is now exposed to the reconciler via a `BackupRegistrar` interface and
  registered on every reconcile when `spec.backup.enabled` is true.
- Scaled-cluster phase is now consistently `Scaling` when replicas change
  (the Milestone-1 change had collapsed it to `Provisioning` under
  envtest, which the regression test caught).

[Unreleased]: https://github.com/MerlionOS/tsdb-operator/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.2.0

## [0.1.0] — 2026-04-13

First tagged release. The operator reconciles a `PrometheusCluster` CRD
into a running Prometheus deployment with HA health checking, scheduled
S3 backups, a REST management API, and an audit log. Verified end-to-end
on kind.

### Added

- `PrometheusCluster` CRD with spec fields for replicas, image, retention,
  storage, resources, S3 backup (cron + bucket + credentials), and
  `remoteWrite` endpoints (`basic_auth` / `bearer_token`).
- Controller that reconciles a headless Service, a ConfigMap containing
  `prometheus.yml`, and a StatefulSet per cluster, with `--web.enable-admin-api`
  gated on `spec.backup.enabled`.
- Finalizer that cleans up metrics series on cluster delete.
- HA health checker (`internal/ha`) — probes `/-/ready` on each replica and
  deletes unhealthy pods to trigger rescheduling.
- Backup scheduler (`internal/backup`) — cron-driven snapshot via the
  Prometheus admin API, upload to S3/MinIO via aws-sdk-go-v2.
- Audit logger (`internal/audit`) — PostgreSQL-backed record of every
  cluster operation.
- REST API (`pkg/api`) — gin server: list/create/get/delete clusters,
  trigger manual backup, query audit log.
- Prometheus metrics — `tsdb_operator_cluster_phase`,
  `tsdb_operator_backup_total`, `tsdb_operator_failover_total`.
- `tsdb-ctl` CLI — `list` and `restore` commands against S3.
- Helm chart at `charts/tsdb-operator/`.
- Grafana dashboard at `grafana/dashboards/tsdb-operator.json`.
- Documentation: README (en/zh), ROADMAP (en/zh), `docs/COMPARISON.{en,zh}.md`,
  `docs/BACKUPS.{en,zh}.md`, `docs/TSDB-LANDSCAPE.{en,zh}.md`, ADRs 0001–0003.
- Test suite: envtest specs for the reconciler, unit tests for HA, Backup,
  audit, REST API, and `renderConfig`.
- CI: lint (golangci-lint), unit tests, envtest, e2e on kind.

[0.1.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.1.0
[0.3.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.3.0
[0.3.1]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.3.1
