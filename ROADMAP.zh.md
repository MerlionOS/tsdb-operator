# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.3.1 — 2026-04-13

在 kind 验证时发现并修复了三个 Thanos sidecar 相关 bug
（`--storage.tsdb.{min,max}-block-duration`、`global:` 块重复、缺
`external_labels`）。推荐 0.3.0 且启用了 `spec.thanos.enabled` 的用户升级。

### v0.3.0 — 2026-04-13

生态主题 release。可选的 Thanos sidecar（`spec.thanos.enabled`），加上
`prometheus-operator ↔ tsdb-operator` 双向迁移指南。

### v0.2.0 — 2026-04-13

Hardening release。REST API 接入 manager + cert-manager TLS；kind 端到端
测试；修复四个真实 bug（`SetupSignalHandler` 被调两次、template 不更新、
scheduler 从未 Register、REST API 从未启动）。

### v0.1.0 — 2026-04-13

首个 tag release。`PrometheusCluster` CRD、reconciler、HA 检查器、S3
备份调度、PostgreSQL 审计日志、gin REST API、Helm chart。

逐 release 明细见 [`CHANGELOG.md`](CHANGELOG.md)。

## 下一个版本 v0.4.0

Milestone 4 还剩两件，v0.4.0 选一（待定）：

- [ ] **`PrometheusClusterSet` CRD。** 跨 namespace 聚合，共享备份 /
  审计策略。旗舰多集群特性。
- [ ] **审计日志保留策略。** `audit_log` 分区表 + 定期清理 + 行数 metric。
  小而自洽。

## 以后

- [ ] **更靠谱的备份产物。** 当前调度器上传的是 admin API 的 JSON
  响应体；真正的 point-in-time 恢复还需要把磁盘上的 snapshot 目录
  tar 打包上传。记录在 [`docs/RESTORE.md`](docs/RESTORE.md) 顶部。
- [ ] **Webhook 校验。** 在 admission 阶段拒绝非法的
  `spec.backup.schedule` cron 表达式，而不是等 cron 触发时才报错。
- [ ] **可组合的抓取配置。** 让用户能在不手改 ConfigMap 的前提下，
  往生成的 `prometheus.yml` 上叠加额外的 `scrape_configs`。

## Non-goals（明确不做的事）

保持边界清晰：

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
