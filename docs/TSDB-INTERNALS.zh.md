# Prometheus TSDB 内部原理：那些影响了 tsdb-operator 设计的细节

English: [TSDB-INTERNALS.en.md](TSDB-INTERNALS.en.md)

面向已经在运维层面熟悉 Prometheus 的读者。讲清楚 TSDB 具体怎么工作，
然后说明 operator 暴露的那些原语（block duration flag、snapshot admin
API、`/-/ready` 探针、资源上限）为什么是给定内部实现下的自然选择。

每一节末尾都有一段 **Design Impact**，指向 repo 里落实这个概念的具体文件。

## 1. TSDB 总览

Prometheus TSDB 运行两层：

- **Head block** —— 内存（加 mmap 尾巴）的写入路径。所有抓取到的样本都
  先进这里。最近约 2 小时的数据。
- **持久化 block** —— 磁盘上不可变的目录。每个 block 包含 chunks、
  倒排索引、元数据文件、tombstone 日志。

```
scrape ──▶ append ──▶ WAL (fsync) ──▶ Head (memSeries + chunks)
                                       │
                                       ▼ 每 2h
                                 compaction 切出一个 block
                                       │
                                       ▼
                              /prometheus/<ULID>/
                                 ├── chunks/000001
                                 ├── index
                                 ├── meta.json
                                 └── tombstones
```

源码：[`prometheus/tsdb/db.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/db.go)、
[`prometheus/tsdb/head.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/head.go)。

**Design Impact。** 直接 `cp -a /prometheus/` 不安全：head 一直在写，
compaction 会改名文件。正确原语是 Prometheus 自带的 snapshot API，
它对已完成的 block 建硬链接 + 可选的 head 快照，拿到一个一致性视图。
所以
[`internal/backup/backup.go:155`](../internal/backup/backup.go) 里是先
POST `/api/v1/admin/tsdb/snapshot`，再 tar 目录上传。

---

## 2. WAL (Write-Ahead Log)

WAL 保证进程重启不丢数据。每次 `Append` 先落 WAL，样本才对查询可见。
记录类型：

- **Series** —— 新的 `(labels) → series ID` 映射
- **Samples** —— `(series ID, timestamp, value)` 三元组
- **Tombstones** —— 删除意图（见 §4）

segment ~128 MiB 轮转。启动时 head 通过重放 WAL + checkpoint（被压缩
过的 WAL segment）重建。

`remote_write` 另有一套**独立**的 WAL，用来在远端不可达时缓冲样本 ——
这是一套独立的 shard-queue 子系统，不是摄入 WAL。源码：
[`prometheus/storage/remote/wal_watcher.go`](https://github.com/prometheus/prometheus/blob/main/storage/remote/wal_watcher.go)。

**Design Impact。** WAL 天然是 per-instance，没有官方支持的跨副本共享
方案。所以 `tsdb-operator` **不**尝试搞自定义的 WAL 共享 HA，而是：

- 跑 `spec.replicas >= 2`，每个 Prometheus 一份 WAL，各自靠自己的 PVC
  保障持久性。
- 查询时用 Thanos Query 去重，或通过 `spec.remoteWrite` 推到持久化
  后端（见
  [`api/v1/prometheuscluster_types.go`](../api/v1/prometheuscluster_types.go)
  的 `RemoteWriteSpec`）。

`remote_write` 自己的 WAL 缓冲正是让 `remoteWrite` 作为 HA 集成点
可行的原因：接收端短暂挂掉不丢数据。

---

## 3. Chunk 存储与 Gorilla 压缩

样本按 chunk 组织，每 chunk 最多 120 个样本（1 分钟抓取间隔对应约
2 小时）。每 chunk 用 Gorilla 编码
（[Facebook 2015 VLDB 论文](https://www.vldb.org/pvldb/vol8/p1816-teller.pdf)）：

- **时间戳**：delta-of-delta —— 存样本间间隔的变化量，稳定抓取时
  几乎总是 0。
- **值**：与前一个值做 XOR，只编码发生变化的 bit 段。

典型压缩：编码后约每样本 1.37 字节。

```
Chunk 生命周期：
  open (head) ──▶ full (120 samples 或 2h) ──▶ mmap 到磁盘
                                              ──▶ 并入 block
```

**Design Impact。** 当 `spec.thanos.enabled` 为 true 时我们会固定
`--storage.tsdb.min-block-duration=2h` **和** `max-block-duration=2h`
（[`internal/controller/prometheuscluster_controller.go:271`](../internal/controller/prometheuscluster_controller.go)）。
两个原因：

1. Thanos sidecar 按 2 小时 block 上传；把 Prometheus 固定在同样的
   节奏下，shipping 行为可预测。
2. Thanos 在对象存储上做自己的下游 compaction —— 如果 Prometheus 先
   本地 compaction，产出的 block Thanos 就没法干净地上传。

retention（`spec.retention`，默认 15 d）是独立的旋钮，Thanos opt-in
**不**会改它。

---

## 4. Compaction

compaction 把小 block 合并成大 block，并物理删除被 tombstone 标记的
series：

- head 切出一个 2 h block。
- planner 把相邻 block 合并，直到合并范围能塞进下一级。
- 默认比例：每一级的最大持续时间跟 retention 联动
  （`max_block_duration = min(retention/10, 31d)`）。15 d retention
  上限 1.5 d block；1 y retention 上限 31 d。源码见
  [`tsdb/compact.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/compact.go)。

tombstone 是惰性的：通过 admin API `DELETE` 只写一条 tombstone 记录，
series 要等到下一次包含该 block 的 compaction 才真正消失。

**Design Impact。** snapshot API 返回的是当前 block 和 head 的**硬链接**。
硬链接共享 inode，所以即使 compaction 在我们上传过程中改写了 block，
我们的 tar 还在读被链接住的目录 —— 文件系统会替我们把 inode 留到
最后一个引用消失。所以
[`internal/backup/backup.go`](../internal/backup/backup.go) 里跑备份
cron 不用跟 compaction 状态协调，挂钟时间触发即可。代价是某次备份
可能还没看到刚切出的 block；下一次 cron 会拿到。

---

## 5. Index

每个持久化 block 自带倒排索引：

```
label name ──▶ label value ──▶ [series IDs] ──▶ [chunk refs]
  __name__       up              [3, 17, 42]     [(0x...), (0x...)]
  job            prometheus      [3, 17]         ...
```

PromQL selector (`{job="prometheus",instance=~"…"}`) 翻译成 posting list
求交。**基数**（label 组合的唯一数量）直接决定 posting list 占多少内存。

> 生产 Prometheus OOM 的头号原因是高基数。常见罪魁：把 user ID、
> request ID、trace ID、原始 URL path 塞进 label。

**Design Impact。** `PrometheusCluster` 的 `spec.resources` 是一等
字段，见
[`api/v1/prometheuscluster_types.go`](../api/v1/prometheuscluster_types.go)
的 `PrometheusClusterSpec.Resources`。用这个 operator 管理 Prometheus
集群的平台应该：

- 保守设置内存 *limit*；被 OOM kill 后通过 WAL 重放能干净恢复。
- 用 VPA / HPA 盯内存使用率而不是 CPU。
- 告警 `prometheus_tsdb_head_series` 的增长率，而不是绝对值。

---

## 6. Head Block

head block 是 `memSeries` 表加当前打开的 chunk。从 Prometheus 2.19
开始 head chunk 是 **mmap'd** 的 —— 磁盘文件是真源，不是内存缓冲。
crash 最多丢失当前这个不完整 chunk 加上最后一次 fsync 之后的 WAL 尾部。

head 的时间范围超过 `chunkRange`（默认 2h）时，compaction 会把范围内
的 series 切成 block 并截断 head。

**Design Impact。**
[`internal/ha/ha.go:81`](../internal/ha/ha.go) 探的是 `/-/ready` 不是
`/-/healthy`：

- `/-/healthy` → 进程还活着。
- `/-/ready` → head 已经跑完 WAL 重放，TSDB 可以响应查询。

刚启动的 Prometheus 可能 healthy 了好几秒还不 ready（WAL 重放耗时
跟样本量成正比）。探 `/-/ready` 才是"这个副本该不该进查询"的正确
信号。未来可以同时探两个端点区分"进程死了应该重建"和"活着但还在
重放应该等"。

---

## 7. Snapshot API

`POST /api/v1/admin/tsdb/snapshot` 生成
`/prometheus/snapshots/<timestamp>-<hash>/`：

- 每个已完成 block：对原始 block 目录建硬链接树。
- 默认把当前 head 也作为新 block 写入（`?skip_head=true` 跳过）。
- 返回 `{"status":"success","data":{"name":"<dirname>"}}`。

一致性：硬链接是原子的，链接到的是不可变（block）或冻结（head）的
内容。无文件损坏风险，不需要跟 compaction 协调。

需要 `--web.enable-admin-api`。operator 把这个 flag 的开关绑在
`spec.backup.enabled` 上
（[`internal/controller/prometheuscluster_controller.go:262`](../internal/controller/prometheuscluster_controller.go)）
—— 不必要的 admin 面始终关着。

完整备份链路：

```
cron 触发
  └─▶ POST /api/v1/admin/tsdb/snapshot          (backup.go:155)
  └─▶ 从响应里解析 "data.name"                   (snapshotResp)
  └─▶ SPDY exec `tar -C /prometheus/snapshots \
        -cf - <name>` 在 <cluster>-0 上         (exec.go)
  └─▶ stdout pipe 到 s3 manager.Upload          (s3client.go)
  └─▶ SPDY exec `rm -rf …/<name>`               (cleanup)
  └─▶ Status().Update(lastBackupTime = now)
```

**Design Impact。** 考虑过但放弃的方案：

- **VolumeSnapshot (CSI)**：备份完整性绑死 CSI 行为，各云厂商不一致，
  而且仍要把数据送出集群才能应对 region 整挂。见
  [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md)。
- **rsync / 原始文件拷贝**：跟 compaction 抢同一批文件 —— 名字和
  inode 都在动。
- **只用 Thanos sidecar**：可行，但强迫用户接整套 Thanos 栈才能有
  备份。我们把 Thanos 留成 opt-in。

备份 cron 只写 S3 对象并打 `status.lastBackupTime`；REST 触发的手动
备份会额外通过
[`pkg/api/server.go`](../pkg/api/server.go) 的 `triggerBackup`
handler 记入 PostgreSQL audit log（cron 路径目前不过审计）。

---

## 8. 总结 —— TSDB 概念如何在这个 repo 里兑现

| TSDB 概念 | tsdb-operator 里怎么体现 |
|---|---|
| WAL per-instance，不能跨副本共享 | 跑 N 副本；查询端 Thanos Query 去重或 `spec.remoteWrite`；不造自定义 WAL 共享。 |
| Snapshot API = 硬链接一致性 | `backup.go` POST API、tar 快照目录、multipart 上传。不用跟 compaction 协调。 |
| head chunk mmap'd；`/-/ready` = WAL 重放完成 | `ha.go` 探 `/-/ready` 而不是 `/-/healthy`。 |
| block 时长独立于 retention | Thanos opt-in 固定 `min/max-block-duration=2h`；`spec.retention` 正交不变。 |
| 基数决定内存 | `spec.resources` 一等字段；文档警告运维盯 series 增长。 |
| tombstone 惰性删除 | 数据要等到 compaction 才物理消失。需要 GDPR 级删除要触发 manual compaction。 |
| `--web.enable-admin-api` 是 snapshot 的前置条件 | 控制器仅在 `spec.backup.enabled` 时追加该 flag。 |

## 进一步阅读

- Fabian Reinartz，[Writing a Time Series Database from Scratch](https://fabxc.org/tsdb/) —— TSDB 原版设计随笔。
- Pelkonen 等，[Gorilla: A Fast, Scalable, In-Memory Time Series Database](https://www.vldb.org/pvldb/vol8/p1816-teller.pdf) —— 本 TSDB 沿袭的压缩论文。
- [Prometheus TSDB 格式规范](https://github.com/prometheus/prometheus/blob/main/tsdb/docs/format/README.md) —— block 目录布局、索引格式。
- [Thanos block 格式](https://thanos.io/tip/thanos/storage.md) —— sidecar 上传的东西。
- [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md) —— 为什么默认选定时 snapshot 而不是持续 `remote_write`。
