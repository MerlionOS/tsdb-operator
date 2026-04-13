# v1.0 准备

English: [V1-PREP.md](V1-PREP.md)

这份文档记录发布 `v1.0` 之前要做的事。v1 不是功能 release —— 而是
我们承诺 `observability.merlionos.org/v1` 在 [semver](https://semver.org/)
意义上稳定的那一刻。

> 如果你正在读这份文档并准备给 CRD 加新字段，问自己一句："我赌得起
> 接下来 12 个月里这个字段名 / 形状不出问题吗？"如果不能，留到 v0.x。

## 为什么是现在

到 v0.10.x，operator 已经覆盖了 Milestone-4 的全部承诺，加上 Later
里几个高价值项。再没明显在拉的 feature。每个 kind 验证轮抓出真实 bug
的 streak（11 个 release 里 10 个）也稳定下来了。该停止加 surface、
开始保证 surface。

## API 稳定性 review

每个 CRD 的每个字段，都需要一个 v1 verdict：

| 字段                                     | v1 状态 | 说明 |
|------------------------------------------|---------|------|
| `PrometheusCluster.spec.replicas`         | **stable** | int32，默认 1，webhook + CRD schema 都强制 min 1 |
| `PrometheusCluster.spec.image`            | **stable** | string，默认指向某个 Prometheus minor |
| `PrometheusCluster.spec.retention`        | **stable** | string，按 Prometheus duration 校验 |
| `PrometheusCluster.spec.storage.size`     | **stable** | resource.Quantity |
| `PrometheusCluster.spec.storage.storageClassName` | **stable** | `*string` |
| `PrometheusCluster.spec.resources`        | **stable** | `corev1.ResourceRequirements`，结构化 |
| `PrometheusCluster.spec.backup.*`         | **stable** | 平铺的 S3 字段；唯一别扭的是 `endpoint`（MinIO 逃生口），保留 |
| `PrometheusCluster.spec.remoteWrite[]`    | **stable** | `URL` 必填；鉴权只走 Secret 引用 —— 故意不支持 inline secret |
| `PrometheusCluster.spec.thanos.*`         | **stable** | enable + image + objstore secret。没有 retention 字段 —— Prometheus 自己管 |
| `PrometheusCluster.spec.additionalScrapeConfigs` | **review** | 当前是 string，见下面 "Breaking-change 清单" |
| `PrometheusCluster.status.*`              | **stable** | phase 是 closed enum；LastBackupTime + LastFailoverTime 是 `*metav1.Time` |
| `PrometheusClusterSet.spec.clusterSelector` | **stable** | `*metav1.LabelSelector` |
| `PrometheusClusterSet.spec.namespaceSelector` | **stable** | `*metav1.LabelSelector` |
| `PrometheusClusterSet.spec.backupTemplate` | **stable** | `*S3BackupSpec`；overlay 规则在 v0.8.0 文档里钉死了 |

## Breaking-change 清单

v1 冻结之前要考虑改的东西：

1. **`additionalScrapeConfigs string` → `additionalScrapeConfigs.inline string` + `additionalScrapeConfigs.secretRef *LocalObjectReference`?**
   v0.9.x 故意只上 inline 形式。如果直接把 string 形式带进 v1，以后想加
   Secret 变体就得用新字段名，最后 forever 留下
   `additionalScrapeConfigs`（string）和 `additionalScrapeConfigsSecretRef`
   （ref）这种丑配对。更干净的做法是 v1 直接上一个结构体，含 `inline`
   和 `secretRef` 两个子字段，webhook 强制互斥。
   **决定：是的，v1 之前重构。**

2. **`status.lastBackupTime` 是 per-cluster 而不是 per-replica。**
   将来跑多副本共享备份时，我们会想要 per-replica 记录。今天没消费者
   想要"每个 replica 的备份时间"，保留现状即可，将来可以加
   `status.replicas[].lastBackup` 而不破坏现有字段。
   **决定：不改。**

3. **`PrometheusClusterSpec.Backup.Endpoint`** 是 MinIO 逃生口，
   严格说不该在用户面 CRD 上。但也没更干净的地方放 —— operator 级别的
   `--s3-endpoint` 有，per-cluster 覆盖只能这么放。
   **决定：保留，文档里说明"用于测试 / 内部对象存储"。**

4. **`spec.thanos.image` 默认指向一个钉死的 Thanos 版本（`v0.36.1`）。**
   v1 cut 时确保默认是当时较新的 Thanos LTS。

5. **`spec.image` 默认值** —— Prometheus 同理。v1 时钉到当时最新的
   2.x。

## Conversion webhook 决策

两条路：

- **路径 A —— 原地升级。** v0.10.x 的 schema 直接成为 v1。v0.10.x 的
  用户升级 operator 即可，CR 不用改。要求**在 v1 之前**就把上面
  `additionalScrapeConfigs` 形状问题解决掉（v1 之后没清理窗口了）。
- **路径 B —— `v1` 和 `v1alpha1` 并存。** 上 conversion webhook，让 v1
  在该 break 的地方 break。工作量大；允许同时改多处；是 operator 真正
  v1 的标准做法。

**推荐路径 A**，前提是 `additionalScrapeConfigs` 重构（项 1）是**唯一**
一处 breaking change。其他 schema 都已经够好。为了一处字段重命名
背 conversion-webhook 的复杂度不划算。

如果 review 过程中又冒出第二处 breaking change，重新评估倾向 B。

## Deprecation 政策（v1 之后）

精神上对齐 [Kubernetes API deprecation 政策](https://kubernetes.io/docs/reference/using-api/deprecation-policy/)：

- 字段废弃要走 `// Deprecated:` godoc + `+kubebuilder:deprecatedversion:warning`，
  保留一个 operator minor version 才能删。
- 重命名时新旧字段并存一个 release；两边都设值时，新字段优先（旧值
  mirror 过来）。
- Schema 删除需要 major bump（v1 → v2）。

## v1 不承诺的事

- 内部包结构（`internal/...`）**不**稳定。从 operator 二进制以外
  import 这些包不被支持。
- REST API JSON shape **不**在 CRD semver 保证范围内 —— 它跟随 CRD
  形状，但可以自由加字段。
- Helm chart values **跟随**变化但可以加新的可选 key 而不需要 major
  bump（增加非破坏）。
- audit log 表结构由 operator 拥有，不允许外部读取。

## v1.0.0 tag 之前的 checklist

- [x] `additionalScrapeConfigs` 重构成结构体 —— **v0.11.0**
- [x] `// +kubebuilder:validation:*` 标记 review：`remoteWrite[].url`
  和 `backup.schedule` 加 MinLength，`retention` 加 Pattern（Prometheus
  duration），`status.phase` 加 Enum
- [x] Storage version：`PrometheusCluster` 和 `PrometheusClusterSet`
  都加了 `+kubebuilder:storageversion`
- [x] Print columns：`kubectl get prometheuscluster` 现在显示
  Phase / Ready / Age；`kubectl get prometheusclusterset` 显示
  Members / Age
- [ ] 默认 `image` 和 `spec.thanos.image` 升版（留到 v1.0 cut 时挑
  fresh 的稳定版本）
- [ ] CHANGELOG 在 `## [1.0.0]` 段只列**有意为之**的 breaking changes
- [ ] v0.x → v1.0 的迁移 note（唯一一次性 CR 改动就是 v0.11.0 的
  `additionalScrapeConfigs` 形状）
