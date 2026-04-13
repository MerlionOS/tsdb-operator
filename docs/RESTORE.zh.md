# 恢复 Runbook

English: [RESTORE.md](RESTORE.md)

从 `tsdb-operator` 写到 S3/MinIO 的备份里把 Prometheus 集群拉回来。

> **备份模型。** v0.6.0 起调度器从 Prometheus Pod 的
> `/prometheus/snapshots/<ts>/` 把磁盘上的快照目录 tar 流式 multipart
> 上传到 S3，然后删除目录释放磁盘。归档里是真实的 TSDB block（chunks、
> index、meta.json），可恢复。详情见
> [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md)。

## 什么时候要恢复

- PVC 被误删或数据损坏。
- Region / AZ 失效，在另一个集群重建。
- 定期演练 —— 证明备份真的能用。

## 前置条件

- 工作站上有 `tsdb-ctl`：
  `go install github.com/MerlionOS/tsdb-operator/cmd/tsdb-ctl@latest`
- 备份 bucket 的凭证已导出为 `AWS_ACCESS_KEY_ID` /
  `AWS_SECRET_ACCESS_KEY`。
- 目标集群有 `PrometheusCluster` CR（可以是空的），因为我们需要
  StatefulSet / PVC 来接收数据。

## Step 1 — 列出可用快照

```bash
tsdb-ctl list \
  --bucket tsdb-backups \
  --prefix demo \
  --endpoint http://minio.example.com   # 真 AWS S3 时省略这个参数
```

输出按时间从新到旧：

```
2026-04-13T00:19:00Z          72  demo/demo/20260413T001900Z.tar
2026-04-13T00:18:00Z          72  demo/demo/20260413T001800Z.tar
```

## Step 2 — 下载快照

默认取最新：

```bash
tsdb-ctl restore \
  --bucket tsdb-backups --prefix demo \
  --endpoint http://minio.example.com \
  --dest ./restore
```

或指定具体的 key：

```bash
tsdb-ctl restore \
  --bucket tsdb-backups \
  --key demo/demo/20260413T001800Z.tar \
  --dest ./restore
```

## Step 3 — 把归档投到目标 Pod

选一个副本作为恢复目标。如果是重建集群，先 apply `PrometheusCluster`
等 StatefulSet 拉起 replica 0。

```bash
# 把归档拷进 Pod
kubectl cp ./restore/20260413T001800Z.tar \
  <ns>/<cluster>-0:/prometheus/

# 解压到 TSDB 目录
kubectl exec -n <ns> <cluster>-0 -- \
  tar -xf /prometheus/20260413T001800Z.tar -C /prometheus/
```

## Step 4 — 重启副本

```bash
kubectl delete pod -n <ns> <cluster>-0
```

StatefulSet 重建 Pod，Prometheus 启动时加载恢复的 TSDB block。验证：

```bash
kubectl -n <ns> port-forward pod/<cluster>-0 9090:9090
curl 'http://localhost:9090/api/v1/query?query=up' | jq '.data.result | length'
```

## Step 5 — 清理

```bash
rm -rf ./restore
```

## 恢复失败时的常见原因

- **`ListObjectsV2: NoSuchBucket`** — 核对 `--bucket` 和 region / endpoint。
- **`tar: unexpected EOF`** — 备份链路写的是 admin API marker 而不是归档。
  按你实际上传的产物形式调整 Step 3，或者先把备份端补齐（见顶部说明）。
- **重启后 Pod CrashLoop** — TSDB 损坏，用 `--key` 指定更老的快照重试。
- **metrics 能查但区间不完整** — Prometheus 只加载完整的 block。确认恢复
  后 Pod 的 `/prometheus/wal/` 是空的，残留的 WAL 会遮住恢复数据。
