# Changelog

All notable changes to this project are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/MerlionOS/tsdb-operator/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/MerlionOS/tsdb-operator/releases/tag/v0.1.0
