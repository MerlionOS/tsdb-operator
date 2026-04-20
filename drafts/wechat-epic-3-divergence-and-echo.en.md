# The Observability Epic · Book III: The Divergence and the Echo

> 中文: [wechat-epic-3-divergence-and-echo.zh.md](./wechat-epic-3-divergence-and-echo.zh.md)

> Three thousand years ago, the East chose the Qintianjian.
> The West chose the Panopticon.
> Three thousand years later, the story repeats.
> And I wrote a small Kubernetes operator — call it one more footnote
> to a very old problem.

---

## Prologue: The Fork in 2013

2013. Two things were happening at the same time.

In Berlin, Julius Volz and Matt Proud's Prometheus had been running at
SoundCloud for half a year and was attracting its first outside
contributors.

In Beijing, DiDi's operations team was building something internally
called **Nightingale** (夜莺) — a monitoring and alerting platform.
It was solving a similar problem to Prometheus's, but with a different
posture. Prometheus was a tool engineers had written for themselves.
Nightingale was a **product designed for operations managers**.

That same year, a US SaaS company called **PagerDuty** was racing
toward IPO. In China, **no company was seriously building an
equivalent** — local teams stitched together DingTalk groups, phone
robots, and SMS gateways.

Nobody imagined that ten years later, around this one core technology,
East and West would grow **two utterly different observability
ecosystems**.

If you've read Book I — this is familiar.

---

## Chapter I · Three Real Questions Behind the Modern Divergence

### Question One: Why Didn't PagerDuty Make It Into China?

PagerDuty is the default in North America for incident response.
Inside China, it barely exists.

The reasons are easy to name:

- International credit-card barriers
- Data sovereignty and export-compliance rules
- Missing integration with DingTalk, WeChat Work, and Feishu —
  **work doesn't happen in Slack**

An empty market fills. Local players rose:

- **Flashduty** (by Flashcat / 快猫星云) — a commercial PagerDuty
  equivalent
- **OpsPilot**, **Jihu IncidentOps** — similar positioning

Here's the interesting part: **these products didn't merely copy
PagerDuty**.

PagerDuty's native alert-deduplication is, frankly, unexceptional.
Domestic products have been much more aggressive about collapsing
"same metric, same service, close time window" into a single alert.
PagerDuty integrates well with Slack; domestic products integrated
**interactive response** into DingTalk bots, WeChat Work groups, and
Feishu cards — acknowledge, reassign, escalate, and resolve alerts
from inside the IM itself. That interaction density is not something
PagerDuty can match.

Domestic IncidentOps also shipped AI earlier — alert summaries,
root-cause hypotheses, LLM-generated incident timelines. Quality may
be uneven, but **they shipped**.

Why ship? Because the Chinese enterprise-software sales channel
tolerates "half-done features going in" in a way SaaS buyers in
Silicon Valley don't. US SaaS is NPS-and-churn-sensitive; AI features
wait until they're genuinely useful.

**This is not a technology difference. It's an organizational one.**

### Question Two: Why Did Pixie Die While DeepFlow Rose?

2018. California. A startup called **Pixie Labs** started building
eBPF application observability — zero-instrumentation, auto-generating
metrics, traces, and logs from the Linux kernel.

Pixie was the hot seed of the moment. "The next generation of APM."

In 2020, New Relic acquired Pixie.

After the acquisition, Pixie faded. Core team members left. Community
contributions slowed. Release cadence stretched from biweekly to
semi-annual. By 2023, Pixie was effectively dead.

In the same window, **DeepFlow** (open-sourced by Yunshan Networks /
云杉网络) climbed the CNCF Landscape. In 2023, DeepFlow joined CNCF
Sandbox. Stars passed 7,000. Contributors expanded beyond China to
Southeast Asia and Europe.

**Same track. Opposite trajectories.**

Pixie took the default Silicon-Valley open-source exit: get acquired,
then vanish. DeepFlow took a different one: stay independent, open
core, layer commercial value on top.

There's a structural feature of the Chinese market here: **the
culture of big companies acquiring open-source projects is much
weaker than in Silicon Valley**. Reasons are layered — the valuation
model for open source isn't mature, integration costs are high, and
**domestic users are warier of big-company capture of open-source
projects**. This gives Chinese observability startups a strange
survival zone: "can't die, can't exit."

Pixie died. The direction didn't. **eBPF observability survived
inside China** and is filling the gap Pixie left, worldwide.

On the same track, **Kindling** (from Harmony Cloud / 谐云) runs in a
similar direction, lighter-weight.

This is one area where **China isn't catching up — it's leading**.

### Question Three: Why Didn't Datadog's Business Model Survive Locally?

A Silicon Valley Series C company pays Datadog **$150,000 a year**
and engineers accept it. What the money buys is "we don't run the
monitoring stack ourselves."

A Chinese company at the same stage — that figure might be **the
entire annual server budget for the company**.

So Chinese teams are extraordinarily sensitive to "dollars per
metric-sample stored."

- **VictoriaMetrics** has much higher penetration in China than in
  Silicon Valley. Its per-sample compression reaches about 0.4 bytes
  (Prometheus default is around 1.3). For budget-constrained teams,
  that's a **structural advantage**
- **TDengine** (by TAOS Data / 涛思数据) bets on IoT and industrial
  time series — the Chinese manufacturing, smart-device, and
  new-energy-vehicle sectors produce a time-series volume that
  Prometheus + VM can't handle
- **GreptimeDB** bets on unified metrics + logs + traces, built on
  Rust and Arrow, cloud-native. It's a track the Silicon Valley
  majors **don't want to build** because it's self-cannibalization
  for Datadog, Grafana, and others

An underrated factor: **Deng Bao** (Multi-Level Protection Scheme /
MLPS), China's national cybersecurity grading regime.

MLPS has **very specific** requirements for observability: audit
logs, operator traceability, data retention. There's no direct
Silicon Valley analog. As a result, in Chinese observability
products, **audit is a first-class citizen**, while Western products
often rely on integrating a third-party log system to satisfy
compliance.

---

## Chapter II · We Already Forked Three Thousand Years Ago

If you read Book I, the three questions above should already sound
familiar.

**Question One** (PagerDuty couldn't enter, IncidentOps localized) —
domestic products integrate deeply with management, face upward, and
treat compliance as default. They look like the **Qintianjian**:
central institution, clean hierarchy, delivering to decision-makers.

**Question Two** (Pixie died, DeepFlow survived) — Pixie ended by
**being absorbed by capital and dissolving**. DeepFlow lived by
**open-source community + commercial layer**. Silicon Valley takes
the acquisition path; China takes the independent-ecosystem path.

**Question Three** (Datadog doesn't work, VM / TDengine / GreptimeDB
rise) — the defining quality of domestic solutions is **low cost,
high density, integrated**. Same gene as "better and cheaper" in
Chinese manufacturing.

The three modern divergences map back into Book I:

| Ancient | Modern |
|---------|--------|
| The Qintianjian — centralized, professional, reporting upward | Nightingale, Flashduty, Aliyun ARMS — management-facing, integrated, compliance-native |
| The Panopticon — observation as power, observer hidden | Datadog, New Relic — SaaS black box, your data watched by one company |
| The Imperial Postal Relay — national-scale distributed messaging | DingTalk, WeChat Work — organization-level IM as alert channel |
| Paul Revere's lanterns — individual, grassroots alerting | Grafana, Prometheus — tools grown from the engineering community |
| Nightingale's Coxcomb — one nurse moving policy with data | Torkel Ödegaard's one-person Grafana fork |
| The Roman Census — periodic full scans | OTel semantic conventions — cross-vendor schema standard |

**The Eastern gene: top-down, centralized, reporting-oriented,
compliance as first-class.**
**The Western gene: bottom-up, distributed, individual-driven,
standards-oriented.**

Three thousand years later, both are still evolving — **each in its
own environment**.

There's no "right" here. These are two different **organizational
environments** producing two different ecosystems.

Chinese observability products feel "management-heavy" not because
engineers don't know better — they live inside organizations where
**upward accountability is stronger**. Silicon Valley observability
products feel "tool-heavy" not because managers lack authority —
engineers there have **more unilateral autonomy**.

**The modern divergence is the ancient gene, echoing.**

---

## Chapter III · China Isn't Ahead Everywhere

Fair is fair.

**AI Copilot — China is catching up.**

Datadog Bits, Grafana Asserts, Elastic AI Assistant — the top Silicon
Valley products can already give you a plausible hypothesis when you
ask "why did p99 latency spike." Most Chinese products are still at
"summarize this alert for me."

That's not a capability gap. It's a **data-and-context gap**. Datadog
can do these things because it sits on cross-product context from
tens of thousands of customers. Domestic products are still building
that corpus.

**OpenTelemetry participation — China is catching up.**

OTel specs are the de facto standard. Core contributors, SIG chairs,
semantic-convention designers — mostly from Silicon Valley and
European majors: Microsoft, Google, Splunk, New Relic. Chinese
corporate participation is rising but nowhere near enough to move
the spec.

This **will hurt**. Every semantic-convention line in OTel is
defining the vocabulary of global telemetry collection for the next
decade. Chinese-specific scenarios (MLPS audit fields, structured IM
payloads in Chinese) haven't been written into the spec. Everyone
downstream will eventually need a translation layer.

**Production maturity for large-scale long-term storage — China is
catching up.**

Thanos and Mimir have a denser base of production case studies in
Silicon Valley. Public Chinese case studies at that scale are rarer
— partly a culture of sharing, partly that the scale **just hasn't
been reached**.

---

## Chapter IV · So What Actually Matters in Selection?

After the long walk, the practical question: a team in 2026 doing
observability selection — how should they think?

**Three decision lines, taken in order:**

### Decision Line One: The Business Shapes the Skeleton

- **SaaS / internet** → Prometheus + VM / Thanos + Nightingale /
  Grafana
- **IoT / industrial / connected vehicles** → take TDengine
  seriously; don't force-fit Prometheus
- **Hybrid / multi-cloud** → be careful with cloud-vendor-tight
  managed services; migration cost will hit you in two years
- **Financial / compliance-sensitive** → self-hosted + audit + MLPS.
  There isn't much choice

### Decision Line Two: Budget Decides Whether You Go Managed

- **< 100 engineers**: self-hosting often costs more than managed.
  But domestic managed options are limited, so the real pattern is
  "open-source self-hosted + part-time ops"
- **100–500**: hybrid is most common. Core on-prem, edge clusters
  managed
- **> 500**: usually self-hosted + dedicated SRE. Managed is
  supplementary

### Decision Line Three: Team Size Decides How Deep You Build

- **Under 10**: don't self-host Thanos or Mimir. You can't run them.
  Single-node Prometheus + backup + managed alerting is enough
- **10–50**: VictoriaMetrics cluster is viable, but **you need one
  person whose job it is to understand operations**
- **50+**: now you can afford Thanos / multi-region / cross-cluster
  aggregation

One sentence: **the complexity of your observability stack must not
exceed the limit of what your team can operate.**

The trap many Chinese teams fall into is "copying the top company's
architecture." Then three people can't keep it running, an alert
fires at 2 AM, and the whole thing goes down for a day.

---

## Chapter V · The Pitch: The Little Qintianjian I Wrote

Time to explain why I'm the one writing this.

Over the past two years I've looked at several companies' self-hosted
Prometheus stacks, and I kept running into a pattern so repetitive I
wanted to scream: **every company was reinventing the same
scaffolding**.

- Pull a StatefulSet, attach a PVC, write a headless Service, set a
  ConfigMap
- Write cron scripts that call Prometheus admin snapshot, `tar` the
  output, `aws s3 cp` the archive
- Health checking, with broken replicas manually `kubectl delete
  pod`'d back into existence
- Change history? K8s Events, which disappear in an hour
- Cross-namespace aggregation? None. Write your own script

None of this is hard. **It's just tedious, and every setup forgets a
different corner.**

So I extracted the pattern into a Kubernetes operator. It's called
**`tsdb-operator`**.

What it does:

- A `PrometheusCluster` CRD — StatefulSet + PVC + Service + ConfigMap
  in one spec
- **Automatic scheduled snapshots to S3 / MinIO**, with a `tsdb-ctl`
  CLI for list and restore
- Replica health checks + automatic eviction and rebuild
- **PostgreSQL-backed audit log** — every cluster change, every
  backup event, retained with a pruner. (This one was forced in by
  MLPS scenarios in China)
- Cross-namespace aggregation via `PrometheusClusterSet`
  (cluster-scoped CRD)
- Optional gin REST API and Grafana dashboard

It is **narrow on purpose**:

- No alerting UI (that's Nightingale, Grafana)
- No APM (SkyWalking)
- No collection layer (DeepFlow)
- No TSDB itself (Prometheus / VictoriaMetrics / GreptimeDB)

It does one thing: **run Prometheus clusters on Kubernetes with
stable operation, backups, and an audit trail**.

In a sense, it's **a small, modern Qintianjian** — full-time
responsibility for keeping a set of Prometheus clusters alive, every
change on the record (like a court chronicle), every snapshot flushed
to object storage (like the postal relay copying documents outward).

Its relationship to most of the domestic solutions in this article
is, by default, **complementary — not competitive**:

- Nightingale needs a metrics backend → tsdb-operator makes that
  backend dependable
- DeepFlow wants to remote-write metrics → into the Prometheus that
  tsdb-operator manages
- You want long-term storage in VictoriaMetrics / GreptimeDB →
  Prometheus remote-write handles it. tsdb-operator doesn't touch
  that layer

If you happen to be a team self-hosting Prometheus (in China or
anywhere else) and you're tired of the scaffolding, here's a quick
pointer:

- GitHub: `MerlionOS/tsdb-operator`
- In that repo, a technical version of this article lives at
  [*Observability Landscape in Mainland China*](https://github.com/MerlionOS/tsdb-operator/blob/main/docs/CHINA-LANDSCAPE.en.md)

Issues welcome. Criticism welcome. **If the criticism is fair, I'll
fix it.**

---

## The Epic's Final Scene: Back to Mount Li

It's time the story came back to where it started.

**771 BCE. Mount Li.**

King You of Zhou, trying to make Bao Si laugh, lit the beacons. The
lords galloped through the night, arrived to find only a woman
laughing. Three years later, the Quanrong came for real. The beacons
blazed again. No lord answered. The Western Zhou fell.

**2026 CE. A desk somewhere in Beijing / Hangzhou / Shenzhen / San
Francisco / Berlin.**

An engineer stares at the fifteenth message to explode across a
Slack alert channel and, with a cold hand, taps "mute this alert."
The alerts don't stop. But they have stopped meaning anything.

Three hours later, production breaks for real.

---

Tools changed for three thousand years.

Copper to electricity. Beacon fire to Prometheus. Lantern to
PagerDuty. Armillary sphere to Grafana.

**The anxiety of information asymmetry has not moved.**

**The mechanics of alert fatigue have not moved.**

**The tension between centralized observation and individual
observation has not moved.**

**The problem of how an observer avoids contaminating what they
observe has not moved.**

When you do observability selection today, you are not just picking
a tech stack. You are deciding, on a three-thousand-year coordinate
system of human observation, **where your organization stands**.

Choose the Qintianjian path: you get a system friendly to central
decision-makers, compliance-native, cleanly hierarchical — at the
cost of engineering autonomy.

Choose the Panopticon path: you get a tool-driven, standardized,
engineer-led system — at the cost of harder upward reporting.

Choose full self-hosting: you are the one lighting the beacons at
Mount Li — free to fire alerts at will, free to bear the cost of
alerts ignored.

Choose full managed: you are the prisoner in the Panopticon — you
don't know how the SaaS on the other side is looking at you, so you
have to assume they always are.

**There is no right answer. Only the price you are willing to pay.**

---

This is the real epic of observability.

Three thousand years, around a single concern:

**In the places you cannot see, before something goes wrong — let
someone tell you.**

From the column of smoke on Mount Li to the next alert push on your
phone — the essence of this act has not moved, not for one minute.

---

**Fin.**

---

### About the Author

Writes code, writes docs, writes stories. For the last two years has
mostly worked on infrastructure, including `tsdb-operator` — a
Kubernetes operator for the life cycle of Prometheus clusters. If
you've taken scars from self-hosting observability stacks, or want
to argue with anything in these three books, reach out.

Repo: [github.com/MerlionOS/tsdb-operator](https://github.com/MerlionOS/tsdb-operator)

---

### Trilogy Index

- **Book I · [The Flame and the Eyes](./wechat-epic-1-flame-and-eyes.en.md)** — 771 BCE to 1858 CE: the Eastern Qintianjian path vs. the Western Nightingale path
- **Book II · [Iron and Lightning](./wechat-epic-2-iron-and-lightning.en.md)** — 1837 to 2019: from Morse to OpenTelemetry
- **Book III · The Divergence and the Echo** (this book) — 2013 to today: the modern Sino-American divergence, and its three-thousand-year echo

### Chinese Editions · 中文版

- **卷一 · [烽火与眼睛](./wechat-epic-1-flame-and-eyes.zh.md)**
- **卷二 · [铁与电](./wechat-epic-2-iron-and-lightning.zh.md)**
- **卷三 · [分叉与回响](./wechat-epic-3-divergence-and-echo.zh.md)**
