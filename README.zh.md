# tsdb-operator

English: [README.md](README.md)

一个管理 Prometheus 集群完整生命周期的 Kubernetes Operator：开通、
扩缩容、高可用、定时备份到 S3 兼容存储、以及运维操作的审计日志。

> PVC 已经有了，为什么还要 snapshot 到 S3？见
> [`docs/BACKUPS.zh.md`](docs/BACKUPS.zh.md)（[English](docs/BACKUPS.en.md)）。
>
> 如何从备份恢复：[`docs/RESTORE.zh.md`](docs/RESTORE.zh.md)（[English](docs/RESTORE.md)）。
>
> 和 Thanos、VictoriaMetrics 的对比见
> [`docs/COMPARISON.zh.md`](docs/COMPARISON.zh.md)（[English](docs/COMPARISON.en.md)）。
>
> 更广的时序数据库横向对比（Prometheus 生态 + 通用 TSDB）：
> [`docs/TSDB-LANDSCAPE.zh.md`](docs/TSDB-LANDSCAPE.zh.md)
> （[English](docs/TSDB-LANDSCAPE.en.md)）。

## 它解决什么问题

在生产环境跑 Prometheus，每个项目都会重新写一遍同样的东西 ——
StatefulSet、PVC、Headless Service、健康检查、快照、异地备份、
谁在什么时候改了什么。`tsdb-operator` 把这些统一封装成一个声明式的
CRD (`PrometheusCluster`) 和一个小而清晰的控制面。

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
