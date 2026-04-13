# tsdb-operator

English: [README.md](README.md)

一个管理 Prometheus 集群完整生命周期的 Kubernetes Operator：开通、
扩缩容、高可用、定时备份到 S3 兼容存储、以及运维操作的审计日志。

> PVC 已经有了，为什么还要 snapshot 到 S3？见
> [`docs/BACKUPS.zh.md`](docs/BACKUPS.zh.md)（[English](docs/BACKUPS.en.md)）。
>
> 如何从备份恢复：[`docs/RESTORE.zh.md`](docs/RESTORE.zh.md)（[English](docs/RESTORE.md)）。
>
> 和 prometheus-operator 双向迁移：
> [`docs/MIGRATION.zh.md`](docs/MIGRATION.zh.md)（[English](docs/MIGRATION.md)）。
>
> 跨 namespace 聚合的 cluster-scoped CRD：
> [`docs/CLUSTERSET.zh.md`](docs/CLUSTERSET.zh.md)（[English](docs/CLUSTERSET.md)）。
>
> 和 Thanos、VictoriaMetrics 的对比见
> [`docs/COMPARISON.zh.md`](docs/COMPARISON.zh.md)（[English](docs/COMPARISON.en.md)）。
>
> 更广的时序数据库横向对比（Prometheus 生态 + 通用 TSDB）：
> [`docs/TSDB-LANDSCAPE.zh.md`](docs/TSDB-LANDSCAPE.zh.md)
> （[English](docs/TSDB-LANDSCAPE.en.md)）。
>
> 影响这个 operator 设计的 Prometheus TSDB 内部原理：
> [`docs/TSDB-INTERNALS.zh.md`](docs/TSDB-INTERNALS.zh.md)
> （[English](docs/TSDB-INTERNALS.en.md)）。

## 它解决什么问题

在生产环境跑 Prometheus，每个项目都会重新写一遍同样的东西 ——
StatefulSet、PVC、Headless Service、健康检查、快照、异地备份、
谁在什么时候改了什么。`tsdb-operator` 把这些统一封装成一个声明式的
CRD (`PrometheusCluster`) 和一个小而清晰的控制面。

## 特性

**集群生命周期**
- `PrometheusCluster` CRD → StatefulSet + headless Service + ConfigMap + PVC
- 基于 finalizer 的清理；phase 上报（`Provisioning` / `Active` / `Scaling` / `Failed`）
- 扩缩容、镜像升级、retention 修改通过完整 pod-template diff 自动下发

**高可用**
- 跨副本定期 `/-/ready` 探活
- 不健康 Pod 被剔除触发重建，更新 `LastFailoverTime` 并写 K8s Event

**备份与恢复**
- Cron → Prometheus admin snapshot → SPDY exec 流 `tar` → S3 multipart 上传
- 上传成功后清理 Pod 上的 snapshot 目录
- `tsdb-ctl` CLI：`list` / `restore`，支持任何 S3 兼容端点（MinIO / AWS 等）

**Thanos sidecar（可选）**
- `spec.thanos.enabled: true` 挂 sidecar，共享 `/prometheus` 数据卷
- 自动加 `--enable-feature=expand-external-labels` + 每 Pod 独立 `replica` 标签
- 对象存储配置通过 Secret 引用

**Remote write**
- `spec.remoteWrite` 生成到 `prometheus.yml`，支持 `basicAuth` / `bearerToken` Secret

**跨 namespace 聚合**
- `PrometheusClusterSet`（cluster-scoped CRD）按 label 跨 namespace 选中集群
- Status 报告成员数、phase 直方图、成员清单

**Admission 校验（可选）**
- Validating webhook 在 `kubectl apply` 时就拒绝坏 `spec.replicas`、缺 `backup.bucket`、
  坏 cron、空 `remoteWrite[].url`
- 通过 Helm values 配置 cert-manager 签发的 TLS

**审计日志（可选）**
- `audit_log` 由 PostgreSQL 支撑，每次 cluster 变更 + 备份事件都落库
- 保留策略（`--audit-retention-days`）+ 定期 pruner

**REST API（可选）**
- 基于 gin：`/api/clusters`、`/api/clustersets`、`/api/clusters/:name/{backup,audit}`
- 支持 cert-manager TLS

**可观测性**
- Prometheus metrics：`tsdb_operator_{cluster_phase,backup_total,failover_total,audit_*}`
- Grafana 面板位于 `grafana/dashboards/tsdb-operator.json`

**打包**
- `charts/tsdb-operator/` Helm chart，每个子系统都有独立 feature flag
- envtest + kind e2e；每个 release 都在真 kind 集群上验证过

## 架构

```
┌───────────────────────────────────────────────────────────────┐
│                         tsdb-operator                         │
│                                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ PrometheusCluster      │   │ HA 健康检查                │   │
│  │ reconciler             │──▶│ (探活 /-/ready，自动剔除)  │   │
│  │ (StatefulSet + SVC)    │   └───────────────────────────┘   │
│  └────────────┬───────────┘                                   │
│               │                                               │
│               ▼                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ 备份调度                │──▶│ S3 / MinIO                │   │
│  │ (cron + admin 快照)    │   └───────────────────────────┘   │
│  └────────────────────────┘                                   │
│                                                               │
│  ┌────────────────────────┐   ┌───────────────────────────┐   │
│  │ REST API (gin)         │──▶│ 审计日志 (PostgreSQL)      │   │
│  └────────────────────────┘   └───────────────────────────┘   │
└───────────────────────────────────────────────────────────────┘
```

## 快速开始

```bash
# 1. 起一个本地集群，装上 CRD 和 operator
make dev

# 2. 创建一个 PrometheusCluster
kubectl apply -f config/samples/observability_v1_prometheuscluster.yaml

# 3. 看它起来
kubectl get prometheuscluster -w
```

## CRD 示例

完整示例见
[`config/samples/observability_v1_prometheuscluster.yaml`](config/samples/observability_v1_prometheuscluster.yaml)。

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: demo
spec:
  replicas: 2
  retention: 15d
  storage:
    size: 20Gi
  backup:
    enabled: true
    bucket: tsdb-backups
    schedule: "0 */6 * * *"
```

## REST API

| Method | Path                              | 说明                         |
|--------|-----------------------------------|------------------------------|
| GET    | `/api/clusters`                   | 列出所有 PrometheusCluster   |
| POST   | `/api/clusters`                   | 创建集群                     |
| GET    | `/api/clusters/:name`             | 查看集群及状态               |
| DELETE | `/api/clusters/:name`             | 删除集群                     |
| POST   | `/api/clusters/:name/backup`      | 手动触发备份                 |
| GET    | `/api/clusters/:name/audit`       | 查询审计日志                 |

写操作请在请求头带上 `X-Operator: <user>`，操作人会被记入审计日志。

## 开发

```bash
# 本地依赖（审计用的 Postgres、备份用的 MinIO、Grafana）
docker compose up -d

# 改完 api/v1 类型后，重新生成 CRD / deepcopy
make generate manifests

# 单测 + envtest
make test

# 基于 kind 的 e2e
make test-e2e
```

### 目录结构

```
api/v1/                   CRD 类型定义
internal/controller/      PrometheusCluster 控制器
internal/ha/              健康检查 + 自动故障切换
internal/backup/          快照 + S3 上传调度
internal/audit/           PostgreSQL 审计日志
pkg/api/                  gin HTTP 服务
config/                   kustomize manifest (kubebuilder)
grafana/dashboards/       Operator 监控面板
```

## Roadmap

见 [`ROADMAP.zh.md`](ROADMAP.zh.md)（[English](ROADMAP.md)）。

## 许可证

Apache 2.0
