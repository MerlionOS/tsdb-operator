# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.6.0 — 2026-04-13

真实备份产物。通过 SPDY exec 把 Prometheus Pod 上的快照目录 tar 流
multipart 上传到 S3，顺便清理 Pod 上的目录。补上项目最大的诚实缺口。

### v0.5.0 — 2026-04-13

`PrometheusClusterSet` cluster-scoped CRD：跨 namespace 聚合、按 phase
统计成员 + REST API。

### v0.4.0 — 2026-04-13

审计日志保留策略。Logger 被 `cmd/main.go` 实例化，加 `Prune` + 定期
pruner，三个新 metric。

### v0.3.1 — 2026-04-13

kind 验证中发现并修复三个 Thanos sidecar bug。

### v0.3.0 — 2026-04-13

可选 Thanos sidecar + prometheus-operator 双向迁移指南。

### v0.2.0 — 2026-04-13

Hardening。REST API 接入 manager + cert-manager TLS；kind 验证暴露的
四个真实 bug 修掉。

### v0.1.0 — 2026-04-13

首个 tag release。核心全部具备。

逐 release 明细见 [`CHANGELOG.md`](CHANGELOG.md)。

## 下一个版本 v0.7.0

- [ ] **Admission webhook。** 在 admission 阶段拒绝非法的
  `spec.backup.schedule` cron 表达式和其他坏 spec，而不是等 cron
  触发时才报错。Validating webhook + Chart 里的 cert-manager 配线。

## 以后

- [ ] **Set 把 `backupTemplate` 自动 overlay 到成员。** v0.5.0 只在 spec
  里记下 template，没有真正改成员 CR。需要一个"谁说了算"的策略（总是
  overlay 还是仅在成员未设置时）以及成员退出该策略的方式。
- [ ] **可组合抓取配置。** 让用户能不手改 ConfigMap 就往生成的
  `prometheus.yml` 上叠加额外的 `scrape_configs`。
- [ ] **跨 Kubernetes 联邦。** 当前 `PrometheusClusterSet` 是跨 namespace
  不是跨 cluster。未来可以加 `PrometheusClusterFederation` 基于
  kubeconfig 做跨集群聚合。

## Non-goals（明确不做的事）

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
