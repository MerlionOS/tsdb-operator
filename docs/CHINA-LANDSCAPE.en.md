# Observability Landscape in Mainland China

中文: [CHINA-LANDSCAPE.zh.md](CHINA-LANDSCAPE.zh.md)

Teams inside China pick observability stacks from a different shortlist than
the one you'd see in English-speaking communities. Alongside the usual
Prometheus / Grafana / Thanos / VM, there's a set of domestic open-source
projects and local cloud vendors that routinely show up in selection docs.

This isn't a "who's best" scorecard. Most of these projects aren't peers of
`tsdb-operator` at all, so a head-to-head table would be misleading. The
two questions this doc answers:

1. **Where does each project sit** in the stack?
2. Is its relationship with `tsdb-operator` **replace** or **complement**?

## 1. Categories at a glance

| Category | Representative projects / services | Role | Relation to tsdb-operator |
|----------|-----------------------------------|------|---------------------------|
| Managed cloud | Aliyun ARMS, Tencent Cloud TMP, Huawei Cloud AOM | Fully managed Prometheus + APM + alerting | Replaces (use managed or self-host, not both) |
| OSS alerting / observability platform | Nightingale (n9e / 夜莺), Erda | Alerting console, dashboards, user/team RBAC | **Complements** (Nightingale needs a metrics backend) |
| APM / tracing | Apache SkyWalking, CAT | Distributed tracing + app performance | Orthogonal (traces ≠ metrics) |
| Full-stack eBPF observability | DeepFlow, Kindling | Zero-instrumentation collection of metrics/traces/logs | **Complements** (DeepFlow remote-writes to Prometheus) |
| Domestic TSDBs | TDengine, GreptimeDB, LinDB | TSDB backend itself | Orthogonal (possible remote-write target) |

## 2. A closer look at each category

### Managed cloud

| | Aliyun ARMS | Tencent Cloud TMP | Huawei Cloud AOM |
|---|---|---|---|
| **Managed Prometheus** | ✅ | ✅ ("Cloud-Native Monitoring TMP") | ✅ |
| **APM / tracing** | ✅ (App Monitoring) | ✅ (APM) | ✅ |
| **Logs** | Integrates SLS | Integrates CLS | Integrates LTS |
| **RUM / frontend** | ✅ | ✅ | Partial |
| **Pricing** | Reported series + usage | Instance tier + sample volume | Metric volume |
| **Open-source interop** | Prometheus exporters, OTel | Same | Same |

All three accept `remote_write` from a self-hosted Prometheus, so in theory
`tsdb-operator` + managed cloud can coexist (self-hosted scraping, managed
long-term storage). In practice most teams go fully one way or the other.

### OSS alerting / observability platform

| | Nightingale (n9e / 夜莺) | Flashcat / Flashduty | Erda |
|---|---|---|---|
| **Origin** | Originally DiDi OSS → now Flashcat (快猫星云) | Commercial product by Flashcat | OSS by Terminus (端点科技) |
| **Form factor** | OSS, self-hosted | SaaS / commercial | OSS, self-hosted |
| **Role** | Alerting + dashboards + user/team console | IncidentOps: alert grouping/dedup + on-call schedules + incident response (PagerDuty-style) | Cloud-native PaaS (DevOps + observability) |
| **Metrics backend** | Prometheus / VictoriaMetrics / TDengine — **doesn't store data itself** | None — ingests alerts from any source | Built-in (ES / ClickHouse) |
| **Alerting** | ✅ core strength, multi-channel | ✅ focused on alert lifecycle | ✅ |
| **License** | Apache 2.0 | Commercial | Apache 2.0 |

Flashcat-the-company ships **two distinct things** — don't conflate them:

- **Nightingale (OSS)** — alerting and observability console, needs a
  metrics backend behind it
- **Flashcat / Flashduty (commercial)** — on-call, alert dedup/routing,
  incident response — a PagerDuty-equivalent, not a metrics platform

**Nightingale is the most natural companion to `tsdb-operator`**: it
doesn't store data, it plugs into a Prometheus / VM data source.
`tsdb-operator` is the thing that keeps that Prometheus cluster healthy
and backed up. Flashduty can sit downstream of Nightingale for on-call /
incident workflows. A typical stack:

```
kube-state-metrics / node-exporter   ──┐
                                       ├──▶  Prometheus managed by tsdb-operator ──▶  Nightingale (queries / alerts / dashboards)
app /metrics                           ──┘
```

### APM / tracing

| | Apache SkyWalking | CAT |
|---|---|---|
| **Origin** | Huawei community → ASF top-level | Dianping / Meituan |
| **Focus** | Tracing + service-mesh observability | Tracing + business monitoring |
| **Storage** | ES / BanyanDB / H2 | HDFS / MySQL |
| **Current activity** | Active | Low |

SkyWalking and Prometheus live on **different axes**: SkyWalking looks at
one request's call chain, Prometheus looks at aggregates. Most teams run
both. `tsdb-operator` only handles the Prometheus side.

### Full-stack eBPF observability

| | DeepFlow | Kindling |
|---|---|---|
| **Origin** | Yunshan Networks (云杉网络) | Harmony Cloud (谐云) |
| **Collection** | eBPF + BPF CO-RE | eBPF |
| **Data** | metrics + traces + network flows | metrics + traces |
| **Output** | Own DB + Prometheus remote-write | Prometheus / OTLP |

eBPF stacks pair with `tsdb-operator` the obvious way: **eBPF collects,
Prometheus stores**. Either scrape DeepFlow agents' `/metrics` via
`PrometheusCluster.spec.additionalScrapeConfigs`, or configure DeepFlow's
remote-write to point at the Prometheus instance tsdb-operator manages.

### Domestic TSDBs

| | TDengine | GreptimeDB | LinDB |
|---|---|---|---|
| **Origin** | TAOS Data (涛思数据) | Greptime (startup) | Ele.me OSS |
| **Primary use** | IoT / industrial time series | Unified metrics / logs / traces | High-throughput metrics |
| **Query** | SQL | SQL + PromQL | Proprietary / partial PromQL |
| **License** | AGPL-3.0 + commercial | Apache 2.0 | Apache 2.0 |

These are TSDBs themselves. Their relationship to `tsdb-operator` is the
same as "VictoriaMetrics and tsdb-operator" — a **remote-write target**
for long-term storage, while Prometheus keeps only a short local window.
They don't replace what tsdb-operator does (as long as you're still
running Prometheus).

## 3. Selection intuition

- **Already on ARMS / TMP / AOM** → skip `tsdb-operator`, stay managed
- **Want full alerting + dashboards + RBAC but self-hosted data** →
  Nightingale + `tsdb-operator`
- **Want zero-instrumentation application data** → DeepFlow / Kindling →
  remote-write to the Prometheus that `tsdb-operator` manages
- **Long-term storage + SQL analytics** → Prometheus (`tsdb-operator`) +
  GreptimeDB / VictoriaMetrics as remote-write target
- **Pure APM needs** (call chains, app perf) → SkyWalking, runs alongside
  `tsdb-operator` on a separate axis
- **Not on Kubernetes** → `tsdb-operator` doesn't apply (it's k8s-native)

## 4. Where this operator sits in the ecosystem

`tsdb-operator` deliberately does one thing: **run Prometheus clusters on
Kubernetes reliably, with backups and an audit trail.** It does not do the
alerting UI (that's Nightingale / Grafana), or APM (SkyWalking), or the
collection layer (DeepFlow), or the TSDB itself (Prometheus / GreptimeDB
/ VM).

So against most of the projects above, the default relationship is
**complement**, not compete. The real competitors are only two:

- Managed Prometheus from a cloud vendor (that's a business decision, not
  a technical one)
- Upstream `prometheus-operator` (migration path:
  [`docs/MIGRATION.md`](MIGRATION.md))
