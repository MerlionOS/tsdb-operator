# Roadmap

English: [ROADMAP.md](ROADMAP.md)

已经交付的、接下来要做的、明确不做的。

## 已交付

### v0.8.0 — 2026-04-13

`PrometheusClusterSet.spec.backupTemplate` 真正投射到成员 CR。成员
通过注解 opt-out；成员自己 `backup.enabled=true` 永远赢。

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

## 下一个版本 v0.9.0

- [ ] **可组合的抓取配置。** 用户面的 `spec.additionalScrapeConfigs`
  （inline YAML 或 Secret 引用），reconciler 合并进生成的
  `prometheus.yml`，不用手改 ConfigMap。今天 operator 自己拥有
  ConfigMap、每次 reconcile 都会覆写，自定义抓取就很别扭，这一项
  把这个坑填了。

## Non-goals（明确不做的事）

保持边界清晰：

- 不重写时序数据库，Prometheus 仍是引擎。
- 不和 Thanos / Mimir / VM 比全局查询。
- 不替代 Alertmanager / `vmalert` 做告警。
- **不做跨 Kubernetes 联邦。** 早期 roadmap 上有过一个住在管理集群里
  跨 kubeconfig 协调的 `PrometheusClusterFederation`。我们决定不做，
  两个理由：（1）这个方向已经被成熟方案占满 —— Karmada、Open Cluster
  Management、Argo CD ApplicationSet —— 真到这个规模的用户大概率已经
  在跑其中之一；（2）老老实实做要跨多个 release（多集群 client 池、
  跨集群 watch、不可达集群降级、per-cluster RBAC），这块工作量会挤占
  垂直方向（更好的抓取配置、更好的备份产物、更好的审计）的时间，
  而后者用户价值更直接。推荐做法：用 Argo CD ApplicationSet 或
  Karmada 把 `PrometheusCluster` / `PrometheusClusterSet` CR 分发到
  多个集群；用 Thanos Query 聚合查询数据。
