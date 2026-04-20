# 中国大陆可观测性方案横向速览

English: [CHINA-LANDSCAPE.en.md](CHINA-LANDSCAPE.en.md)

国内团队在选可观测性栈时，候选项和国外社区常见组合不太一样 ——
除了 Prometheus / Grafana / Thanos / VM 这些通用方案，还有一批
国产开源项目和本地云厂商托管服务会进到选型表里。

这份文档**不是"谁强谁弱"的打分表**。绝大多数国产方案和 `tsdb-operator`
不是同类产品，硬比不公平。这里要回答的是两个问题：

1. 他们分别**站在哪一格**？
2. 和 `tsdb-operator` 是**替代关系**还是**共存关系**？

## 1. 分类速览

| 类别 | 代表项目 / 服务 | 角色 | 和 tsdb-operator 的关系 |
|------|----------------|------|------------------------|
| 云厂商托管 | 阿里云 ARMS、腾讯云 TMP、华为云 AOM | 全托管 Prometheus + APM + 告警 | 替代（上了托管就不自建） |
| 开源告警/观测平台 | 夜莺 Nightingale (n9e)、Erda | 告警中台、仪表盘、用户/权限 | **互补**（夜莺需要一个 metrics 后端） |
| APM / tracing | Apache SkyWalking、CAT | 分布式追踪 + 应用性能 | 正交（trace ≠ metrics） |
| 全栈 eBPF 观测 | DeepFlow、Kindling | 零侵入采集 metrics/traces/logs | **互补**（DeepFlow remote-write 到 Prometheus） |
| 国产时序数据库 | TDengine、GreptimeDB、LinDB | TSDB 后端本身 | 正交（可作为 remote-write 目的地） |

## 2. 各类细看

### 云厂商托管

| | 阿里云 ARMS | 腾讯云 TMP | 华为云 AOM |
|---|---|---|---|
| **Prometheus 托管** | ✅ | ✅（云原生监控 TMP） | ✅ |
| **APM / tracing** | ✅（应用监控） | ✅（APM） | ✅ |
| **日志** | 集成 SLS | 集成 CLS | 集成 LTS |
| **前端监控** | ✅ | ✅ | 部分 |
| **定价模式** | 按 reported series + 按量 | 按实例规格 + samples | 按指标量 |
| **开源组件** | 接 Prometheus exporter、OTel | 同 | 同 |

说明：这三家都支持把自建 Prometheus 的数据**remote write** 过去，
所以理论上 `tsdb-operator` + 云托管是可以混合的（边缘自建采集、
云端做长期存储和查询），但多数团队不会这么折腾，要么全托管要么全自建。

### 开源告警 / 观测平台

| | 夜莺 Nightingale (n9e) | Flashcat / Flashduty | Erda |
|---|---|---|---|
| **出处** | 原滴滴开源 → 快猫星云（Flashcat） | 快猫星云商业产品 | 端点科技开源 |
| **形态** | 开源自建 | SaaS / 商业企业版 | 开源自建 |
| **定位** | 告警 + 仪表盘 + 用户/团队中台 | IncidentOps：告警聚合降噪 + 值班排班 + 事件响应（类 PagerDuty） | 云原生 PaaS（DevOps + 观测） |
| **metrics 后端** | Prometheus / VictoriaMetrics / TDengine，**自己不存数据** | 不存数据，对接任何告警源 | 自带（基于 ES/ClickHouse） |
| **告警** | ✅ 强项，多通道 | ✅ 专注告警生命周期 | ✅ |
| **License** | Apache 2.0 | 商业 | Apache 2.0 |

快猫星云一家同时做**两件事**，别混淆：

- **夜莺（开源）** —— 告警和观测中台，要接一个 metrics 后端
- **Flashcat / Flashduty（商业）** —— 告警之后的值班、聚合、降噪、协作
  （对标 PagerDuty），不负责采集或存储

**夜莺是 `tsdb-operator` 最自然的搭档**：夜莺不存数据，它要接一个
Prometheus/VM 数据源；`tsdb-operator` 就是把那个 Prometheus 集群跑稳、
备份好的部分。Flashduty 可以接在夜莺的告警输出后面做值班侧的事。
典型组合：

```
kube-state-metrics / node-exporter   ──┐
                                       ├──▶  tsdb-operator 管的 Prometheus ──▶  夜莺（查询 / 告警 / 面板）
应用 /metrics                          ──┘
```

### APM / tracing

| | Apache SkyWalking | CAT |
|---|---|---|
| **出处** | 华为社区 → ASF 顶级 | 大众点评 / 美团 |
| **重点** | tracing + service mesh observability | tracing + 业务监控 |
| **存储** | ES / BanyanDB / H2 | HDFS / MySQL |
| **当前活跃度** | 活跃 | 低 |

SkyWalking 和 Prometheus 是**不同维度**：SkyWalking 看单个请求的调用链，
Prometheus 看聚合指标。大多数团队两个都跑。`tsdb-operator` 只管
Prometheus 那一路。

### 全栈 eBPF 观测

| | DeepFlow | Kindling |
|---|---|---|
| **出处** | 云杉网络 | 谐云 (Harmony Cloud) |
| **采集方式** | eBPF + BPF CO-RE | eBPF |
| **数据** | metrics + traces + 网络流 | metrics + traces |
| **输出** | 自带 DB，也支持 Prometheus remote-write | Prometheus / OTLP |

eBPF 方案和 `tsdb-operator` 的组合方式是：**eBPF 做采集，Prometheus 做存储**。
`PrometheusCluster.spec.additionalScrapeConfigs` 抓 DeepFlow agent 的
`/metrics`，或者在 DeepFlow 侧配 remote-write 到 tsdb-operator 起的
Prometheus。

### 国产时序数据库

| | TDengine | GreptimeDB | LinDB |
|---|---|---|---|
| **出处** | 涛思数据 | Greptime（创业公司） | 饿了么开源 |
| **主场景** | IoT / 工业时序 | metrics/logs/traces 融合 | 高写入 metrics |
| **查询** | SQL | SQL + PromQL | 自有 / PromQL（部分） |
| **License** | AGPL-3.0 + 商业版 | Apache 2.0 | Apache 2.0 |

这三个都是 TSDB 本身，和 `tsdb-operator` 的关系等价于 "VictoriaMetrics
和 tsdb-operator" —— 可以作为 **remote-write 后端**存长期数据，
Prometheus 本地只留短窗口。不会替代 tsdb-operator 的集群生命周期管理
（前提是你还在用 Prometheus）。

## 3. 选型直觉

- **已经全上了 ARMS / TMP / AOM** → 不用 tsdb-operator，继续托管
- **想要完整告警 + 面板 + 用户体系**，但数据自建 → 夜莺 + tsdb-operator
- **要零侵入拿应用数据** → DeepFlow / Kindling → remote-write 到
  tsdb-operator 的 Prometheus
- **看重长期存储 + SQL 分析** → Prometheus (tsdb-operator) + GreptimeDB
  / VictoriaMetrics 作为 remote-write 目的地
- **纯 APM 诉求**（调用链、应用性能） → SkyWalking，和 tsdb-operator
  各跑各的
- **不跑 Kubernetes** → tsdb-operator 不适用（它是 k8s 原生的）

## 4. 这个 operator 在生态里的位置

`tsdb-operator` 故意只做一件事：**在 Kubernetes 上把 Prometheus 集群
跑稳、备好份、审计好**。它不做告警 UI（那是夜莺/Grafana 的事），
不做 APM（那是 SkyWalking 的事），不做采集层（那是 DeepFlow 的事），
也不做 TSDB 本身（那是 Prometheus / GreptimeDB / VM 的事）。

所以和上面列的大多数国产方案，**默认的关系是互补**，不是竞争。
真正的竞争对手只有两个：

- 云厂商的托管 Prometheus（用不用托管是个业务决策，不是技术决策）
- 上游的 `prometheus-operator`（迁移路径见
  [`docs/MIGRATION.zh.md`](MIGRATION.zh.md)）
