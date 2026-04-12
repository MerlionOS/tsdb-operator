# 时序数据库横向对比

English: [TSDB-LANDSCAPE.en.md](TSDB-LANDSCAPE.en.md)

在给 `tsdb-operator` 或者更广的可观测性栈选后端时，常见的时序数据库速览。
分成两组：**Prometheus 生态（PromQL 兼容）** 和 **通用时序数据库**
（数据模型不同，作参考）。

## 1. Prometheus 生态（PromQL 兼容）

| | Prometheus | Thanos | Cortex | Grafana Mimir | VictoriaMetrics |
|---|---|---|---|---|---|
| **出处** | Prometheus 社区 (2012) | Improbable → CNCF | Weaveworks → CNCF（已归档） | Grafana Labs（从 Cortex fork） | VictoriaMetrics Inc. |
| **扩展模型** | 单节点（纵向） | sidecar + 对象存储扇出 | 微服务，水平分片 | 微服务，水平分片 | `vmstorage` 分片 + 复制 |
| **长期存储** | 本地 TSDB（受 retention 限制） | 对象存储 (S3/GCS/Azure) | 对象存储 | 对象存储 | 本地磁盘（集群版）或 `vmbackup` 到 S3 |
| **查询语言** | PromQL | PromQL（Thanos Query） | PromQL | PromQL | MetricsQL（PromQL 兼容超集） |
| **多租户** | 无 | 有限 | ✅ 一等公民 | ✅ 一等公民 | ✅（集群版） |
| **HA / 去重** | 双副本 | Thanos Query 去重 | 多副本 ingester | 多副本 ingester | `-replicationFactor` |
| **典型写入** | 10 万–100 万 samples/s/节点 | 取决于底层 Prometheus | 1000 万+/s（横向） | 1000 万+/s（横向） | 1000 万+/s，同组中密度 / 内存最优 |
| **许可证** | Apache 2.0 | Apache 2.0 | Apache 2.0（已归档） | AGPL-3.0 | Apache 2.0（核心），企业功能 BSL |
| **托管服务** | AWS AMP、GCP、Grafana Cloud | Red Hat OpenShift、社区 | （已归档） | Grafana Cloud | VictoriaMetrics Cloud |

说明：
- **Cortex** 对新部署来说基本被 **Grafana Mimir** 取代；Mimir 就是
  从 Cortex fork 出来、专门解决运维痛点的。
- **Thanos vs Mimir/VM** 是一个大的架构分歧：Thanos 把 Prometheus 当摄入
  引擎、只加读路径；Mimir 和 VM 自己跑 ingester。

## 2. 通用时序数据库

| | InfluxDB 3 (IOx) | TimescaleDB | QuestDB | TDengine | ClickHouse | M3DB |
|---|---|---|---|---|---|---|
| **存储引擎** | Apache Parquet + Arrow（列式） | PostgreSQL + hypertable | 列式，mmap | 自研列式 | 列式（MergeTree） | 自研，M3TSZ 压缩 |
| **查询语言** | InfluxQL + SQL（历史 Flux） | SQL（Postgres 方言） | SQL | SQL | SQL | M3 Query / 经 M3Coordinator 支持 PromQL |
| **主要场景** | 可观测性 + IoT | 在 Postgres 内做可观测性 + 分析 | 低延迟金融 / IoT | IoT / 工业 | 通用分析（实际常当 TSDB 用） | Uber 级别 metrics |
| **扩展模型** | 云原生 + 对象存储 | 纵向 + 多节点集群 | 单节点（横向 WIP） | 集群 | 分片集群 | 带复制的分片集群 |
| **高基数** | 非常高（列式） | 高 | 中等 | 高 | 非常高 | 高 |
| **典型写入** | 百万级 samples/s（集群） | 10 万–100 万/s/节点 | 百万级/s 单节点 | 百万级/s 集群 | 1000 万+/s 集群 | 1000 万+/s 集群 |
| **许可证** | MIT（核心 v3）/ 商业云 | Apache 2.0 + 部分 TSL | Apache 2.0 | AGPL-3.0 + 商业 | Apache 2.0 | Apache 2.0 |
| **托管服务** | InfluxDB Cloud | Timescale Cloud、AWS 托管 | QuestDB Cloud | TDengine Cloud | ClickHouse Cloud、Altinity | 无（Uber 内部） |

说明：
- **InfluxDB 3 (IOx)** 是用 Rust + Parquet 完全重写的版本，和 1/2 代差别
  很大。看到旧评测基本都不适用了。
- **ClickHouse** 严格说不是 TSDB，但大量场景当 TSDB 用（Uber M3 → ClickHouse
  迁移、SigNoz、PostHog）。
- **TimescaleDB** 的强项是"**同一个库里，时序数据和关系数据能 SQL 关联**"。

## 选型直觉

- **"我就想 Prometheus 在 K8s 上稳定跑"** → Prometheus + `tsdb-operator` +
  Alertmanager。超出规模再加 Thanos。
- **"需要跨 20+ 个 Prometheus 的全局查询"** → Thanos 或 Mimir。
- **"同样 PromQL-ish API，想要每块钱最高的写入量"** → VictoriaMetrics 集群。
- **"要 SQL、要和业务表 join"** → TimescaleDB。
- **"要做一个把高基数 tracing / events 和 metrics 一起上的产品"** →
  ClickHouse 或 InfluxDB 3。
- **"IoT 设备集群，边缘受限"** → TDengine 或 QuestDB。

## 这个 Operator 和它们正交

`tsdb-operator` 管的是**Prometheus 进程的生命周期**。长期存储后端怎么选
（Thanos vs Mimir vs VM vs 只做 S3 snapshot）是另一个独立决策 —— 选
运维复杂度匹配你团队规模、查询模式匹配用户实际需要的那个。
