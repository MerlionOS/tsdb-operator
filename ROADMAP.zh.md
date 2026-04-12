# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.1.0 — 2026-04-13

首个打了 tag 的 release。operator 能开出 Prometheus 集群、对副本做
探活与故障切换、按 cron 快照到 S3，通过 REST API 提供管理能力并记录
审计日志。已在 kind 上端到端验证通过。

完整列表见 [`CHANGELOG.md`](CHANGELOG.md)。

## 下一个版本 v0.2.0

下个 release 想带上的东西。未打勾的是开放工作项。

- [ ] **REST API 加 TLS。** 集成 cert-manager，在 operator service 层面
  终结 TLS，不只靠 ingress。
- [ ] **在 kind+MinIO 上端到端验证备份链路。** 当前备份代码路径有单测
  但还没有实际对着一个真的对象存储跑过一次完整回环。
- [ ] **e2e 测试覆盖 scale / delete / failover 场景**（替换占位 e2e）。
- [ ] **`tsdb-ctl restore` 端到端文档 + 演示。** CLI 写好了，但配套
  runbook 还没写。

## Milestone 4 — 多集群与生态

每个都是一个相对独立的大块，可以各自带起一个 0.x 版本。

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
