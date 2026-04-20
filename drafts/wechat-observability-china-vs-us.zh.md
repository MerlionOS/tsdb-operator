# 可观测性的中美分叉：国内走的，其实不是同一条路

> 从 PagerDuty、Pixie、Datadog，到夜莺、DeepFlow、GreptimeDB ——
> 如果你以为国内可观测性只是在"抄硅谷"，你可能漏掉了正在发生的事。

---

## 一、一个你可能没注意到的分叉点

2023 年，一个在硅谷被寄予厚望的开源项目 **Pixie** 被 New Relic 收购之后慢慢
停摆了。Pixie 走的是 eBPF + 零侵入自动观测，当年被看作"下一代 APM"的种子。
被收购之后，核心团队散了，社区活跃度一路下滑。

就在同一段时间，国内的 **DeepFlow**（云杉网络开源）在 CNCF Landscape 里
慢慢爬到了 7k+ stars，拿到了进 CNCF Sandbox 的资格，商业化也跑起来了。

**同一条赛道，硅谷凉了的项目，国内在升温。**

这不是一件孤立的事。

- PagerDuty 在国内几乎打不进来，于是出现了 **Flashduty**、OpsPilot、极狐
  这样一批本土 IncidentOps 产品
- Datadog 一年 15 万美元的账单让硅谷团队咬牙都得用，国内团队听完会先愣
  三秒再说"我们自建吧"——于是 **VictoriaMetrics** 在国内的密度远高于硅谷
- SkyWalking 明明是 Apache 顶级项目，但在北美 APM 市场几乎存在感为零；
  在国内，它是**默认选项**之一

这些不是巧合。

## 二、两张典型栈，放一起看看

**硅谷一家 500 人工程团队的典型栈：**

```
采集      Prometheus + OpenTelemetry SDK + Datadog Agent
存储      Datadog (managed) / Grafana Cloud / 自建 Mimir
告警      Datadog / PagerDuty
trace     Datadog APM / Jaeger
日志      Datadog Logs / Loki / Splunk
```

**国内一家 500 人工程团队的典型栈：**

```
采集      Prometheus + SkyWalking Agent + DeepFlow eBPF
存储      自建 Prometheus + VictoriaMetrics / ClickHouse
告警      夜莺 Nightingale (自建) → Flashduty (值班)
trace     SkyWalking (自建)
日志      ElasticSearch (自建) / ClickHouse / 阿里云 SLS
```

一眼看下去，感觉是"同一个功能清单，换了一批 logo"。

但如果你真在团队里做过选型，你会发现**决策路径完全不一样**。
硅谷的选型是 "Datadog 贵，但一站式，买就完事了"。国内的选型是
"Datadog 我们不考虑（你懂的），那么自建、自建到什么程度、哪块用开源、
哪块上云厂商托管？"

**这个选型起点的差异，才是所有分叉的源头。**

## 三、三个真问题，让国内生态长出了不同的形状

### 问题一：PagerDuty 进不来，IncidentOps 长出了本土生态

PagerDuty 是北美事件响应的默认选项，但在国内几乎没有存在。原因很简单：
国际支付门槛、数据合规、钉钉/企业微信/飞书集成缺位。

空白市场必定有人来填。于是出现了：

- **Flashduty**（快猫星云）—— PagerDuty 的对标产品，主打告警聚合、值班排班、
  事件时间线
- **OpsPilot**、**极狐 IncidentOps** —— 类似定位的创业产品

有意思的是：这些产品不是单纯的"中国版 PagerDuty"。**它们比原版做得更细。**

- 告警降噪：PagerDuty 本体的降噪能力比较弱，国内产品普遍把相同指标、
  相同服务、相近时间窗的告警聚合得更狠
- IM 深度集成：钉钉机器人、企业微信群、飞书卡片的**交互式响应**——在 IM
  里直接确认、转派、升级，这个交互密度 PagerDuty 做不到
- AIOps：国内产品很早就在塞 LLM 告警摘要、根因推荐。效果先不说，**起码
  敢上**

为什么敢上？因为国内企业软件的销售路径允许"半成品先进去"。硅谷 SaaS
更在意 NPS 和 churn，AI 功能要等到真有用才发。这是**组织层面的差异**，
不是技术层面的。

### 问题二：Pixie 凉了，但 eBPF 的赌注还在下

Pixie 当年被视为"下一代 APM"的代表作。New Relic 收购之后，核心团队出走，
社区失去推动力。eBPF 可观测性在硅谷开源圈里陷入一段尴尬期 ——
Cilium 在网络层很强，但应用观测这块一直没长出新的领袖项目。

这个位置在国内被 **DeepFlow**（云杉网络）接了过去。

DeepFlow 的赌注是：**用 eBPF 做应用级全栈观测，完全零侵入，metrics、traces、
logs、network flow 一个探针搞定**。它的想法其实和 Pixie 重合度很高。但是：

- Pixie 被大厂收购后消失，DeepFlow 作为独立创业公司还在推
- DeepFlow 选择了**开源 + 商业双轨**，代码 Apache 2.0，核心在 GitHub 上
- 和 Prometheus / Grafana / OTel 都有适配，不是封闭系统

**这不是"中国版 Pixie"**。DeepFlow 已经是全球 eBPF 可观测性赛道上少数有
竞争力的项目之一。你在 KubeCon 欧洲场也能看到他们的 talk。

同赛道还有 **Kindling**（谐云），也是 eBPF 方向，更轻量一点。

这一块**国内没有在追，而是在领**。

### 问题三：预算敏感 + 等保，让 VictoriaMetrics 和 TDengine 更香

硅谷一家 C 轮公司愿意花 15 万美元/年买 Datadog，因为这笔钱换的是
"工程师不用自己维护监控栈"的时间。国内一家同规模公司，这笔钱可能是
全公司一年的**服务器预算**。

于是国内团队对"每块钱能存多少 metrics"非常敏感。

- **VictoriaMetrics** 在国内的渗透率远高于硅谷。VM 单节点的压缩密度能做到
  每 sample 0.4 字节左右，比 Prometheus 默认的 1.3 字节压缩很多。对预算
  敏感的团队，VM 是一个**结构性优势**
- **TDengine**（涛思）押 IoT 和工业时序。国内制造业、智能设备、新能源车
  这些场景的时序数据量级非常大，Prometheus + VM 的方案在这种场景下并不
  合适。TDengine 的写入性能和压缩比是它能成立的根本
- **GreptimeDB** 赌的是"metrics + logs + traces 一体化"，Rust + Arrow 的
  栈，云原生架构。还年轻，但赌的方向是硅谷大厂不太愿意下场做的（因为
  对 Datadog、Grafana 都是自我革命）

还有一个被低估的推力：**等保**（网络安全等级保护）。等保对可观测性的审计
日志、用户操作记录、数据留存有具体要求。这在硅谷完全没有对应的监管场景。
所以你会看到国内的可观测性产品里，**审计**常常是一等公民，而硅谷产品
经常是靠集成第三方日志系统来满足。

## 四、但国内确实也在追的地方

说句公平话。**不是所有方向国内都在领。**

### AI Copilot 落后半拍

Datadog Bits、Grafana Asserts、Elastic AI Assistant —— 硅谷的大厂
可观测性产品已经把 LLM 助手做到"问它'最近为什么 p99 飙高'能给出合理
猜测"的程度。国内产品很多还停在"告警摘要帮你写一段话"的阶段。

这不是能力问题，是**数据和上下文的问题**。Datadog 能做这些是因为它有
几万家客户的跨产品上下文训练。国内产品还在积累这个语料。

### OpenTelemetry 生态的参与度

OTel 是全球可观测性的事实标准。其核心贡献者、SIG 主席、spec 设计者
绝大部分来自硅谷、欧洲的大厂（Microsoft、Google、Splunk、New Relic）。
国内公司参与度在涨，但还远没到能左右 spec 的程度。

这个**会吃亏**：OTel 的 semantic conventions 每一条都是在定义未来几年
全球采集的"方言"，国内场景（比如等保审计字段）没有进到 spec 里，后面
就得自己转。

### 长期存储的工程成熟度

Thanos、Mimir 这类方案在硅谷的生产部署密度还是更高。国内团队做大规模
长期存储的经验案例，公开分享的并不多——部分是因为分享文化不同，部分
是因为规模真的没到那个程度。

## 五、那选型到底该看什么？

绕了一圈，回到实际问题：一个国内团队今天要做可观测性选型，到底该怎么看？

**三条决策线，建议按顺序过：**

### 决策线一：业务形态决定骨架

- **SaaS / 互联网业务** → Prometheus + VM / Thanos + 夜莺 / Grafana。
  这是已经跑出来的主流路线
- **IoT / 工业 / 车联网** → 严肃考虑 TDengine，不要硬塞 Prometheus。
  数据模型就不匹配
- **混合云 / 多云** → 谨慎选云厂商强绑定的托管方案（ARMS / TMP / AOM），
  哪怕便宜。迁移成本会在三年后打脸
- **金融 / 合规敏感** → 自建 + 审计 + 等保合规，基本没得选

### 决策线二：预算决定要不要买托管

- 工程团队 **< 100 人**：自建成本比托管高。但国内托管选项有限，所以
  往往只能"自建开源 + 运维兼职"
- 工程团队 **100–500 人**：混合路线。核心业务自建，边缘或临时集群上
  阿里云 ARMS 或腾讯云 TMP
- 工程团队 **> 500 人**：基本都是自建 + 专职 SRE。托管只作为补充

### 决策线三：团队规模决定自建深度

- **10 人以下团队**：别自建 Thanos / Mimir，跑不动。Prometheus 单机
  + 备份 + 托管告警，够了
- **10–50 人**：可以上 VictoriaMetrics 集群，但要有一个懂运维的人专门
  兜底
- **50 人以上**：才有本钱玩 Thanos / 多 region / 跨集群聚合这些

一句话收束：**可观测性栈的复杂度不该超过团队能运维的上限。**
国内很多团队踩的坑是"按头部公司的架构抄作业"，结果自己三个人维护
不起来。

## 六、私货时间：为什么我做了 tsdb-operator

写到这里，总得交代一下为什么我会琢磨这些。

过去两年我帮一些公司看过 Prometheus 自建栈，看到一个重复得让人痛苦
的现象：**每家公司都在重写同一套脚手架**。

- 拉 StatefulSet、挂 PVC、写 headless Service、配 ConfigMap
- 写 cron 脚本调 Prometheus admin snapshot，`tar` 打包，`aws s3 cp` 上传
- 健康检查，坏副本得手动 `kubectl delete pod` 重建
- 变更记录？靠 K8s Event，一小时后就查不到
- 跨 namespace 聚合？没有，自己写脚本扫

这些东西不是什么技术难点，就是**琐碎且容易忘一个角**。我把它们抽出来
封装成了一个 Kubernetes Operator，叫 `tsdb-operator`：

- 一个 CRD `PrometheusCluster`，帮你把 StatefulSet、PVC、Service、
  ConfigMap 一次性生好
- 自带**定期快照到 S3 / MinIO** 的 cron 调度，以及 `tsdb-ctl` 命令行用来
  list 和 restore
- 副本健康检查 + 自动剔除重建
- **PostgreSQL 审计日志**（每次集群变更、每次备份都有记录，带 retention），
  这条是国内等保场景倒逼我加的
- 跨 namespace 聚合用的 `PrometheusClusterSet`（cluster-scoped CRD）
- 可选的 gin REST API 和 Grafana 面板

它故意做得**很窄**：不做告警 UI（那是夜莺、Grafana 的事），不做 APM
（那是 SkyWalking 的事），不做采集层（那是 DeepFlow 的事），不做
TSDB 本身。它就是把"**在 k8s 上稳定跑 Prometheus 集群**"这一件事做到
可以交付。

所以它和本文提到的大多数国产方案，**默认关系是互补**：

- 夜莺需要一个 metrics 后端？tsdb-operator 就是把那个后端起好的部分
- DeepFlow 要 remote-write metrics 出来？往 tsdb-operator 管的 Prometheus 写
- 想把数据长期存到 VictoriaMetrics / GreptimeDB？Prometheus 配 remote-write
  就行，tsdb-operator 不碰那一层

如果你在国内团队自建 Prometheus，正好又嫌那些脚手架烦，可以看一眼：

- GitHub：`MerlionOS/tsdb-operator`
- 项目里专门写了一篇《[中国大陆可观测性方案横向速览](#)》，本文是
  公众号版，文档版讲得更技术一些

欢迎提 issue，欢迎骂 —— 骂得有道理的我改。

---

## 最后一句

可观测性不是一个纯技术选择，是一个**组织决策**。

国内外的分叉，本质上是**组织环境和市场环境的分叉**：付费意愿不同、
合规环境不同、IM 和协作工具不同、工程团队规模分布不同。

硅谷不是标准答案，国内也不是落后版本。

两条路线在各自的环境里都在演化出自己的最优解。看清楚这一点，
你做的选型才有意义。

---

**关于作者**

在做 `tsdb-operator`（Kubernetes 上管 Prometheus 集群生命周期的 Operator）
和一些别的基础设施项目。如果你在自建可观测性栈踩了坑，或者对上面的
观点想骂两句，欢迎来聊。
