# PrometheusClusterSet

English: [CLUSTERSET.md](CLUSTERSET.md)

`PrometheusClusterSet` 是一个 **cluster-scoped** 资源，按 label 把跨
namespace 的 `PrometheusCluster` 聚合到一起。它本身不创建 Prometheus
实例 —— 只观察成员关系并（可选）携带成员可继承的默认策略。

## 为什么需要它

真实平台里，"可观测性团队"运维一组分布在各产品 namespace 下的 Prometheus
集群。没有上层对象的话，列出它们、看每个 phase、给一个统一默认备份策略
都得在外围拿脚本拼。`PrometheusClusterSet` 就是那个上层对象。

## 示例

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusClusterSet
metadata:
  name: tier1
spec:
  clusterSelector:
    matchLabels:
      tier: t1
  namespaceSelector:
    matchLabels:
      env: prod
  backupTemplate:
    enabled: true
    bucket: tsdb-tier1-backups
    schedule: "0 */4 * * *"
```

会选中所有 label `tier=t1` 的 `PrometheusCluster`，且只在 label `env=prod`
的 namespace 里。

## Status

reconciler 把成员关系和按 phase 的直方图写进 Set 的 status：

```yaml
status:
  memberCount: 3
  phaseCount:
    Active: 2
    Provisioning: 1
  members:
    - namespace: team-a
      name: prom-a
      phase: Active
    - namespace: team-b
      name: prom-b
      phase: Active
    - namespace: team-c
      name: prom-c
      phase: Provisioning
```

任何匹配的 `PrometheusCluster` 变化时，Set 会重新 reconcile（通过
`Watches` 把每个 PrometheusCluster 事件 enqueue 到所有 Set —— Set 数量
很小所以没事）。

## REST API

| Method | Path                          | 说明                  |
|--------|-------------------------------|-----------------------|
| GET    | `/api/clustersets`            | 列出全部 Set          |
| GET    | `/api/clustersets/:name`      | 看某个 Set 的详情      |

## 这一版**不**做的事

- **自动把** `backupTemplate` patch 进匹配到的 `PrometheusCluster`。字段
  在 spec 里记下了，但成员 CR 还得自己写 `spec.backup`。自动 overlay 留到
  下一版做；这样用户能看到策略意图，但 Set 不会悄悄改用户的 CR。
- 跨 Kubernetes 集群联邦。仅限单个 Kubernetes 集群。
