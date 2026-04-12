# 迁移指南

English: [MIGRATION.md](MIGRATION.md)

在 `tsdb-operator` 和 `prometheus-operator` 之间双向迁移的做法。两者
关注点**不同** —— 抓取配置 vs 集群生命周期 —— 所以大多数情况下正确答案
是两个都跑而不是替换。本指南覆盖的是确实需要替换的情况。

## 开始之前

先读
[ADR-0001](adr/0001-why-tsdb-operator-separate-from-prometheus-operator.md)
理解 scope 差异。如果你真正的需求是"我想要声明式抓取配置"，留在
prometheus-operator。如果需求是"我想要 HA、定时 S3 备份、审计、
围绕集群生命周期的 REST API"，**加上** tsdb-operator 就行，不用替换。

## 通用前置步骤

1. 找出源端 CR，dump 到 YAML：
   ```bash
   kubectl get -A prometheuses.monitoring.coreos.com -o yaml > /tmp/from-po.yaml
   kubectl get -A servicemonitors.monitoring.coreos.com -o yaml > /tmp/from-po-sm.yaml
   ```
2. 动手前先把当前 TSDB 异地备份：
   ```bash
   kubectl exec -n <ns> <prometheus-pod> -- \
     curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot
   # 然后把 /prometheus/snapshots/<ts>/ 拷出 Pod
   ```
3. 停掉往集群里写数据的 scraper（`remote_write` 源、pushgateway），
   避免切换过程中有新样本落进来。

## prometheus-operator → tsdb-operator

当你明确想要生命周期 / HA / 备份 / 审计这一层、并且愿意重写抓取配置
时这么做。

### Step 1 — 两个 operator 同时装

```bash
helm install tsdb-operator ./charts/tsdb-operator \
  -n tsdb-operator --create-namespace
```

两个 operator 可以共存，CRD 不冲突（`Prometheus` vs `PrometheusCluster`）。

### Step 2 — 翻译 `Prometheus` CR

常见字段对应关系：

| prometheus-operator (`Prometheus`)    | tsdb-operator (`PrometheusCluster`)               |
|---------------------------------------|---------------------------------------------------|
| `spec.replicas`                       | `spec.replicas`                                   |
| `spec.image`                          | `spec.image`                                      |
| `spec.retention`                      | `spec.retention`                                  |
| `spec.resources`                      | `spec.resources`                                  |
| `spec.storage.volumeClaimTemplate`    | `spec.storage.size` + `storageClassName`          |
| `spec.remoteWrite[].url` + 鉴权        | `spec.remoteWrite[].url` + `basicAuth` / `bearerToken` secret |
| `spec.thanos`                         | `spec.thanos.enabled` + `objectStorageConfigSecretRef` |
| `spec.serviceMonitorSelector`         | *不支持* —— 抓取配置是用户侧的事          |
| `spec.podMonitorSelector`             | *不支持*                                          |
| `spec.ruleSelector`                   | *不支持* —— 直接在 `prometheus.yml` 里写 recording rule |

### Step 3 — 自己管抓取配置

`tsdb-operator` 管着 `prometheus.yml` 所在的 ConfigMap，装完会塞一个
默认值。把你原来的 `ServiceMonitor` / `PodMonitor` 规则翻译成原生
Prometheus scrape config 写进这个 ConfigMap。operator 会保留默认 key
**以外**你做的修改。

另一种做法：只留 prometheus-operator 管 `ServiceMonitor`，把它
`promtool`-生成的配置当成输入。这就是 ADR-0001 里说的"两个都跑"方案。

### Step 4 — 关掉旧的 prometheus-operator 管的那个 Prometheus

```bash
kubectl -n <ns> delete prometheus <name>
```

只会删掉那个 Prometheus StatefulSet，`ServiceMonitor` / `PodMonitor`
/ `PrometheusRule` CR 都留着，自己决定要不要清。

### Step 5 — 从快照恢复（可选）

如果前置步骤做了 TSDB 备份：

```bash
kubectl cp /path/to/snapshot.tar <ns>/<new-cluster>-0:/prometheus/
kubectl exec -n <ns> <new-cluster>-0 -- tar -xf /prometheus/snapshot.tar -C /prometheus/
kubectl delete pod -n <ns> <new-cluster>-0
```

完整流程见 [`RESTORE.zh.md`](RESTORE.zh.md)。

## tsdb-operator → prometheus-operator

当你明确需要 `ServiceMonitor` / `PodMonitor` 作为主要抓取配置接口、
或者要整套 kube-prometheus-stack 时这么做。

### Step 1 — 装 prometheus-operator

```bash
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace
```

### Step 2 — 翻译 `PrometheusCluster` CR

| tsdb-operator (`PrometheusCluster`)  | prometheus-operator (`Prometheus`) |
|--------------------------------------|------------------------------------|
| `spec.replicas` / `image` / `retention` / `resources` | 同名                      |
| `spec.storage.size`                  | `spec.storage.volumeClaimTemplate.spec.resources.requests.storage` |
| `spec.remoteWrite`                   | `spec.remoteWrite`                 |
| `spec.thanos.enabled`                | `spec.thanos` 区块                 |
| `spec.backup`                        | *原生没有* —— 换用 velero / vmbackup，或继续只让 tsdb-operator 跑备份 |
| `spec.backup` + audit log            | *原生没有*                         |

### Step 3 — 写 `ServiceMonitor` CR

迁过来的全部意义就在这一步。把原来 `prometheus.yml` 里每块 scrape
配置翻成 `ServiceMonitor` 或 `PodMonitor`。

### Step 4 — 关掉 tsdb-operator（或者不关）

如果只是迁了 Prometheus 本身，但还想要备份和审计，可以留 tsdb-operator
的 REST API 接另一个 `PrometheusCluster`。不留的话：

```bash
helm uninstall tsdb-operator -n tsdb-operator
kubectl delete ns tsdb-operator
kubectl delete crd prometheusclusters.observability.merlionos.org
```

### Step 5 — 数据恢复

和正向一样：把快照拷进新 Pod，解压到 `/prometheus`，重启。

## 常见故障

- **同一个工作负载出现两套 Prometheus 副本。** 两个 operator 都在跑
  而你没删旧的。`kubectl get pods -l app.kubernetes.io/name=prometheus -A` 确认。
- **Grafana 数据源突然有空洞。** 忘了恢复快照，或者新 PVC 用了不同的
  StorageClass。`kubectl get pvc -n <ns>` 看一下。
- **`ServiceMonitor` 的抓取静默挂了。** 迁到 tsdb-operator 后没把抓取
  规则搬到 `prometheus.yml` 里。补上，或者两个 operator 并跑。
