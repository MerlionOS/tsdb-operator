# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.7.0 — 2026-04-13

Validating admission webhook。非法 `spec.replicas`、缺 `backup.bucket`、
坏 cron、空 `remoteWrite[].url` 在 `kubectl apply` 时就被拒。Helm 经
cert-manager 签发 TLS。

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

## 下一个版本 v0.8.0

- [ ] **把 Set 的 `backupTemplate` 自动 overlay 到成员。** v0.5.0 在 spec
  里记下了 template 但没有真正改成员 CR。做完这个，`PrometheusClusterSet`
  就从"看板"变成真正的策略对象。
  - 策略：仅当成员 `spec.backup.enabled` 未设置 / false 时 overlay；
    成员显式设置的字段始终赢。
  - 退出机制：成员注解
    `observability.merlionos.org/clusterset-opt-out: "true"`。
  - Scope：Set reconciler 里新增一次 Set→成员的投影，冲突检测，
    owner-reference 决策（不 re-parent，只打 label），envtest 覆盖。

## 以后

相对 v0.8.0 更小的候选，还没编进 release：

- [ ] **可组合的抓取配置。** `spec.additionalScrapeConfigs`（inline YAML
  或 secret ref）合并进生成的 `prometheus.yml`，不用手改 ConfigMap。
  剩下几项里用户价值最高；中 scope。
- [ ] **跨 Kubernetes 联邦。** 未来的
  `PrometheusClusterFederation` 按 kubeconfig 聚合多个
  `PrometheusClusterSet`。剩下几项里最大的：需要多集群 client 管理、
  认证、跨集群 watch。可能要拆两个 release。

## Non-goals（明确不做的事）

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
