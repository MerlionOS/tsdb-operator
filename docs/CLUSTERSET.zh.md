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

## `backupTemplate` overlay（v0.8.0 起）

当 Set 带 `spec.backupTemplate` 时，Set reconciler 按下面的顺序把它
覆盖到匹配的成员上：

1. **必须有 template。** `spec.backupTemplate` 为 nil 时啥都不做。
2. **opt-out 最高。** 成员带注解
   `observability.merlionos.org/clusterset-opt-out: "true"` 时永远
   不动它。
3. **成员显式声明胜出。** 成员 `spec.backup.enabled` 已经是 `true`
   的保留自己的配置。
4. **否则**：成员的 `spec.backup` 被整体替换成 template（`enabled:
   true`），并打上注解
   `observability.merlionos.org/clusterset: <set-name>` 便于追溯。

`PrometheusCluster` reconciler 通过自己的 watch 感知到变更，备份
scheduler 在下一次 reconcile 时注册 cron。

### 说明

- **按成员整体取舍，不做字段级合并**（Go 零值和"未设置"不用指针
  包装就没法区分）。成员要么整体继承，要么整体自治。
- 删除 Set **不会**撤销已经 overlay 的配置。这是故意的 —— 删 Set
  时悄悄关掉备份比留着跑危险得多。
- 成员事后把 `enabled` 改成 `true` 等于把所有权要回来：下一次 Set
  reconcile 就不再动它。

## 这一版**不**做的事

- 跨 Kubernetes 集群联邦。仅限单个 Kubernetes 集群。
