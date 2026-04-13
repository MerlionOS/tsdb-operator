# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.5.0 — 2026-04-13

多集群聚合。`PrometheusClusterSet` cluster-scoped CRD 按 label 把跨
namespace 的 `PrometheusCluster` 聚合起来；status 报告成员 + 按 phase
的直方图。REST API + envtest + kind 验证过。

### v0.4.0 — 2026-04-13

审计日志硬化。Logger 终于被 `cmd/main.go` 实例化；加了 `Prune` + 定期
pruner、三个新 metric、Helm chart 通路。

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

## Milestone 4 — 完结 ✅

M4 四件事全部交付：Thanos sidecar（v0.3）、迁移指南（v0.3）、审计保留
策略（v0.4）、`PrometheusClusterSet`（v0.5）。

## 以后

待定，还没编进某个 release。

- [ ] **Set 把 `backupTemplate` 自动 overlay 到成员。** v0.5.0 只在 spec
  里记下 template，没有真正改成员 CR。需要一个"谁说了算"的策略（总是
  overlay 还是仅在成员未设置时 overlay）以及成员退出该策略的方式。
- [ ] **更靠谱的备份产物。** 当前调度器上传的是 admin API JSON；真正
  point-in-time 恢复还需要把磁盘上的 snapshot 目录 tar 打包上传。
  记录在 [`docs/RESTORE.md`](docs/RESTORE.md) 顶部。
- [ ] **Admission webhook。** 在 admission 阶段拒绝非法的
  `spec.backup.schedule` cron，而不是等 cron 触发时才报错。
- [ ] **可组合抓取配置。** 让用户能不手改 ConfigMap 就往生成的
  `prometheus.yml` 上叠加额外的 `scrape_configs`。
- [ ] **跨 Kubernetes 联邦。** 当前 `PrometheusClusterSet` 是跨 namespace
  不是跨 cluster。未来可以加 `PrometheusClusterFederation` 基于
  kubeconfig 做跨集群聚合。

## Non-goals（明确不做的事）

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
