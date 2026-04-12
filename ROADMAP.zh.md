# Roadmap

English: [ROADMAP.md](ROADMAP.md)

接下来要做的事，按意图分组。每组内的顺序是建议的执行顺序，不是硬性排期。

## Milestone 1 — 先让它真的能跑起来

当前脚手架能编译、CI 过，但真的 `kubectl apply` 会挂。这些是首先要修的
实际缺陷。

- [x] **挂载 Prometheus 配置文件。** 当前 StatefulSet 引用
  `/etc/prometheus/prometheus.yml` 但没有挂载对应 ConfigMap → CrashLoop。
  需要：operator 给每个 `PrometheusCluster` 生成默认 ConfigMap 并挂载，
  同时允许用户通过 `spec.configMapRef` 覆盖。
- [x] **启用 admin API 以支持快照。** `spec.backup.enabled: true` 时给容器
  加上 `--web.enable-admin-api`。否则 snapshot 接口 404，备份静默失败。
- [x] **把 HA 和 Backup 控制器注册进 manager。** 现在 `cmd/main.go` 里
  只注册了 `PrometheusCluster` reconciler，`internal/ha` 和
  `internal/backup` 写好了但从未启动。通过 `mgr.Add(...)` 带 flag
  （`--enable-ha`, `--enable-backup`）注册进去。
- [x] **加 finalizer。** 删除 `PrometheusCluster` 时清理 headless Service，
  以及（可选）触发最后一次备份。否则会留下孤儿资源。

## Milestone 2 — 可观测 + 可测试

- [x] **暴露 Prometheus metrics。** 注册 Grafana 面板里已经引用的：
  - `tsdb_operator_cluster_phase{cluster,phase}`
  - `tsdb_operator_backup_total{cluster,result}`
  - `tsdb_operator_failover_total{cluster}`
- [x] **给 reconciler 加 envtest 测试。** 覆盖 create / scale / delete
  和 phase 转换。
- [x] **给 HA 和 Backup 加单测。** 用假 HTTP server + 假 S3 `Uploader`，
  断言 `LastFailoverTime` / `LastBackupTime` 被正确更新。
- [x] **REST API 合约测试。** 用假 client 起 gin router，每条路由跑一遍。

## Milestone 3 — Day-2 打磨

- [x] **`tsdb-ctl restore` CLI。** 从 S3 把快照拉回 PVC（备份的对称动作）。
- [x] **Helm chart。** `charts/tsdb-operator/`，包含 operator 安装和
  Postgres / S3 secret 的 values。
- [x] **`remote_write` 集成。** 可选的 `spec.remoteWrite`，让被管理的
  Prometheus 可以推到 Thanos / Mimir / VictoriaMetrics。
- [ ] **REST API 加 TLS。** 集成 cert-manager，在 operator service 层面
  终结 TLS，不只靠 ingress。
- [x] **`docs/adr/` 下写 ADR。** 记录为什么这个 operator 和 prometheus-operator
  并存、为什么选定时 snapshot 而不是持续 remote-write 等关键决策。

## Milestone 4 — 多集群 / 生态

这一阶段偏愿景，要等 Milestone 1–3 都稳了再做。

- [ ] **跨集群聚合 CRD。** `PrometheusClusterSet` 跨多个 namespace、
  共享备份和审计。
- [ ] **Thanos sidecar 可选开关。** `spec.thanos.enabled: true` 挂一个
  sidecar + objstore config secret。
- [ ] **审计日志保留策略。** 分区表 + 定期清理。
- [ ] **operator 之间的迁移指南。** prometheus-operator ↔ tsdb-operator
  双向迁移。

## Non-goals（明确不做的事）

保持边界清晰：

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
