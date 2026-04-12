# 为什么要把 Prometheus TSDB 定时 snapshot 到 S3？

English: [BACKUPS.en.md](BACKUPS.en.md)

PVC 本身**不是备份**。下面是 `tsdb-operator` 把定时 S3 snapshot 做成
`PrometheusCluster` CRD 一等公民的原因。

## PVC 为什么不够用

**1. PVC 的故障域很小**
大多数 StorageClass（EBS、GCE-PD、local-path）的 PV 是**单可用区**的。
AZ 挂了，数据跟着挂。节点磁盘损坏、PVC 被误删，结果一样 —— PVC 有副本
不等于有异地备份。

**2. Prometheus retention 是**本地**的有界窗口**
`--storage.tsdb.retention.time=15d`，到期就删。合规场景常要求
**保留 ≥1 年**的原始数据，本地块存储放不下，也不经济。

**3. 块存储贵，对象存储便宜**
EBS gp3 大约 $0.08/GB/月；S3 Standard 大约 $0.023/GB/月；Glacier 更便宜。
只把**热数据**留在 PVC、**冷数据**归档到 S3，成本差距很明显。

**4. 灾难恢复和人为误操作**
CR 被误删、retention 被调小、ransomware、region 整个挂 ——
需要一份**和集群解耦**的副本，任何新集群都能拿来恢复。

## snapshot 具体是怎么做的

Prometheus 自带一个 admin API：

```
POST /api/v1/admin/tsdb/snapshot
```

它会在 `/prometheus/snapshots/<时间戳>/` 下**硬链接**当前 TSDB block —
瞬间完成，block 被改写前不占额外磁盘，也不阻塞写入。

`tsdb-operator` 的 backup 控制器干的就是：

```
cron tick
  → POST /api/v1/admin/tsdb/snapshot
  → tar 快照目录
  → PutObject 到 S3 / MinIO
  → 更新 status.lastBackupTime
```

## 和 Thanos 的差别

Thanos 的模型不一样：**sidecar 持续把 2h block 上传到对象存储**，所以
历史数据天然就在 S3，根本没有"备份"这个独立动作。代价是你要跟每个
Prometheus 一起维护 Thanos Sidecar、Store Gateway、Compactor 一整套组件。

`tsdb-operator` 的立场：如果你想继续用**原生 Prometheus**、不引入 Thanos
那套东西，但又需要"集群外有一份、能恢复"—— 定时 snapshot 就是最轻的方案。

## 什么时候 snapshot 不是正确答案

- 你要**无限期保留 + 全部能用 PromQL 查**  → 用 Thanos 或 VictoriaMetrics。
- 你要**跨集群全局查询视图**  → 用 Thanos Query 或 VM。
- 你要**分钟级以内的 RPO**  → 用 `remote_write` 流式写到持久化后端，
  不要指望 cron snapshot。
