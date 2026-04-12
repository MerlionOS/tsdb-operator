# tsdb-operator vs prometheus-operator vs Thanos vs VictoriaMetrics（中文版）

English version: [COMPARISON.en.md](COMPARISON.en.md)

给正在评估 TSDB 方案的同学一个速览。

## 一句话总结

| | tsdb-operator | prometheus-operator | Thanos | VictoriaMetrics |
|---|---|---|---|---|
| **它是什么** | 聚焦**集群生命周期**（开通 / HA / 备份 / 审计）的 Operator | 聚焦 Prometheus 与 Alertmanager **抓取配置声明化**的 Operator | 架在 Prometheus **之上**的水平扩展 + 长期存储层 | 独立时序数据库（单机版 + 集群版） |
| **主要产物** | `PrometheusCluster` CRD + 控制器 | `Prometheus` / `Alertmanager` / `ServiceMonitor` / `PodMonitor` / `PrometheusRule` / `Probe` 等 CRD | Sidecar / Store Gateway / Query / Compactor / Receiver | `vmstorage` / `vminsert` / `vmselect`（或单二进制） |
| **存储模型** | Prometheus 本地 TSDB + PVC，定时 snapshot 到 S3 | Prometheus 本地 TSDB + PVC（存储留给用户解决） | Prometheus 本地 TSDB + 对象存储存历史 block | 自研 TSDB 格式，本地磁盘（集群版分片） |
| **查询语言** | PromQL | PromQL | PromQL（通过 Thanos Query） | MetricsQL（PromQL 兼容超集） |
| **高可用策略** | 多副本 StatefulSet + `/-/ready` 探活 + 剔除故障 Pod | `spec.replicas ≥ 2`，去重留给 Thanos / 远端存储 | 双副本 Prometheus + Thanos Query 去重 | `-replicationFactor` 跨 `vmstorage` |
| **全局视图** | 不内置 | 不内置（配合 Thanos / VM） | ✅ Thanos Query 扇出 | ✅ 集群版 + `vmagent` remote_write |
| **备份** | 一等公民：CRD 直接写 cron → S3/MinIO | 无 | 隐式（历史 block 在对象存储） | 专用工具 `vmbackup` / `vmrestore` |
| **审计日志** | 内置（PostgreSQL） | 无 | 无 | 无 |
| **管理 API** | gin REST：集群 CRUD + 备份 + 审计 | 无（kubectl / Helm） | 无（kubectl / Helm） | 无（kubectl / Helm） |
| **许可证** | Apache 2.0 | Apache 2.0 | Apache 2.0 | Apache 2.0 |

## tsdb-operator 的实际定位

**它是生命周期控制器，不是 TSDB**。底层跑的仍然是上游 Prometheus；
它解决的是大多数团队在每个项目里都要重新造一遍的 "Day 2" 问题：
声明式的 HA、定时异地备份、操作审计。

## 各自适合什么场景

### 选 `tsdb-operator` 的场景

- 想继续用**原生 Prometheus**，只需要把生命周期（开通、扩缩容、HA、
  异地备份、审计）自动化。
- 需要一个**管理用的 REST API**，给控制面 / 内部 UI 用。
- 单集群 / 单 region 规模，不需要全局查询。
- 合规要求留下每一次运维操作的审计轨迹。

### 选 prometheus-operator 的场景

- 你真正想要的是**声明式抓取配置** —— `ServiceMonitor` / `PodMonitor` /
  `PrometheusRule` 这类 CRD。
- 你想要社区事实标准、生态最广（kube-prometheus-stack、仪表盘、社区告警规则）。
- 集群生命周期 / 备份 / 审计这一层你已经有别的东西在管。

> **注意**：prometheus-operator 和 tsdb-operator 是**互补关系**。
> prometheus-operator 管的是"抓什么"，tsdb-operator 管的是
> "Prometheus 集群本身怎么开通、怎么备份、谁改过什么"。
> 实际平台里完全可以两个一起跑。

### 选 Thanos 的场景

- 需要**跨多个 Prometheus 实例的全局查询视图**。
- 想要**无限期的历史数据**存在对象存储里，且仍然用 PromQL。
- 已经在跑 Prometheus，想往上加能力而不是换掉它。
- 关心降采样（downsampling）与多租户。

### 选 VictoriaMetrics 的场景

- 想在同样硬件上拿到比 Prometheus **更高的写入吞吐 / 更低的内存与磁盘**。
- 能接受换一个 TSDB（MetricsQL 兼容 PromQL 但不完全一致）。
- 想要一套同源工具链：`vmagent` / `vmalert` / `vmauth` / `vmbackup`。
- 集群版的分片模型比 Thanos 的 sidecar + store 拆分更贴近你的规模。

## 它们其实可以组合用

这三者并不是非此即彼：

- `tsdb-operator` + **prometheus-operator**：tsdb-operator 负责把 Prometheus
  集群开出来，prometheus-operator 的 `ServiceMonitor` / `PodMonitor` 负责
  驱动抓取配置。
- `tsdb-operator` + **Thanos**：operator 管生命周期，Thanos Query 提供
  跨多个被管 Prometheus 的全局视图。
- `tsdb-operator` 管理的 Prometheus 可以 `remote_write` 到
  **VictoriaMetrics** 做长期存储，本地保留 HA / 备份 / 审计这一层。

## 边界：这个 Operator **不做**什么

- 不提供全局查询 / 联邦层 —— 用 Thanos Query 或 VM 集群版。
- 不做降采样 —— Prometheus 原生没有，配合 Thanos Compactor。
- 不做多租户 label 强制 —— 用 Cortex / Mimir / VictoriaMetrics 集群版。
- 不做告警 pipeline —— 配合 Alertmanager / vmalert。

目标是一个**小而清晰、专注生命周期**的 Operator，而不是把整个可观测性栈
重新实现一遍。
