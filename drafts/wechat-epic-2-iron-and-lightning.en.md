# The Observability Epic · Book II: Iron and Lightning

> 中文: [wechat-epic-2-iron-and-lightning.zh.md](./wechat-epic-2-iron-and-lightning.zh.md)

> From the thirty letters that jumped through a telegraph wire between
> Washington and Baltimore in 1844,
> to a commit pushed to GitHub from a Berlin office in 2012 —
> this is a hundred and eighty years in which humanity swapped the
> tools for "seeing the distance,"
> **from iron and copper, to electricity and silicon**.

---

## Prologue: The Green Screen at Chain Home

Summer, 1940. The southeast coast of England.

A woman named Joan Leaman sat in front of an instrument called a PPI
— a Plan Position Indicator. It was a circular green phosphor screen
not quite half a meter across. Every few seconds, a fan-shaped beam
swept around it, and occasionally a small dim dot would flicker on
and off.

Those dots were German bombers flying across the English Channel.

Joan wore a headset and reported each dot's bearing, range, and
estimated altitude to the operations room of RAF 11 Group. The
operations room used her calls to move wooden blocks across a plotting
table, and from there scrambled Spitfires and Hurricanes against the
incoming raid.

That summer, the Royal Air Force — outnumbered roughly **two to one**
— held off the German air assault, guided by the Chain Home radar
chain.

The Chain Home PPI is **the direct ancestor of the modern monitoring
panel**.

The request_latency curve on your Grafana dashboard, if you trace its
DNA back far enough, leads to those green screens on the Dover coast
in 1940.

Book II starts here.

---

## Chapter I · Electricity Arrives

### 1837: Signal Finally Outpaces the Horse

September 2, 1837. A lab at New York University.

Samuel Morse and Alfred Vail stood at opposite ends of a 500-meter
copper wire. Morse tapped a small key: dot-dot-dot, dash-dash-dash,
dot-dot-dot. A string of electrical pulses ran down the wire and
tripped an electromagnet on the other end, embossing matching marks
onto a paper tape.

The first telegraph.

The weight of this moment sits on top of two thousand years of
background. Before the telegraph, the upper bound on information
transmission was **the horse** — or the beacon fire, which carried
only a few bits of warning. The electrical pulse running down a copper
wire broke that limit by three orders of magnitude in a single step.

Seven years later, on May 24, 1844, Morse sent the first long-distance
public telegraph from the Capitol in Washington to Baltimore. The
message was a line his niece had chosen from the Book of Numbers:

**"What hath God wrought."**

She could not have known that this sentence would echo as the opening
line of every technological revolution for the next hundred and eighty
years.

Twenty years later, a transatlantic submarine cable finally held.
After the permanent connection of 1866, messages between London and
New York dropped from "three weeks by steamer" to "ten minutes by
wire."

**For the first time in history, humanity had real-time global
monitoring.**

London bankers, drinking coffee in the morning, could see last night's
closing gold prices in New York. The British Foreign Secretary could
cable the ambassador in Washington within hours of a crisis breaking
out.

When you pull Prometheus metrics across regions today and the data
arrives in seconds across ten thousand kilometers, you think of it as
a new capability. It isn't. Only the medium is new — copper, radio,
optical fiber — all answers to the same question: **how does here know
there?**

### 1913: The Signal Lights of the Ford Shop Floor

Detroit. Highland Park. The Ford Model T assembly line.

Henry Ford did something that changed industrial history: he split car
assembly into **84 stations**, one worker per station, and ran the
frames from station to station on a moving belt.

The system was efficient enough that Model T assembly time dropped
from 12 hours to 93 minutes.

But line efficiency came with a cost: **if any one station breaks, the
whole line stops**.

So Ford's engineers built an in-shop monitoring system:

- Each station had a **three-color signal light** overhead: green
  (normal), yellow (trouble but still running), red (stop the line)
- Any worker could pull a rope to **actively trigger a yellow or red
  alert**
- The shop supervisor walked the floor, watched the lights, and ran
  toward any red

Toyota later developed this into the **Andon system**, still a
cornerstone concept of lean manufacturing.

Unpack the design:

- Each subsystem self-reports its health
- State is expressed in a **unified visual language**
- A triggered alert has a clear receiver
- **Any node in the workflow can raise an alert**

This was 1913. Today you would call those four principles **health
checks + visual dashboard + on-call routing + developer-triggered
alerting**.

What Ford's engineers invented in a Detroit shop would, a century
later, reappear as a `:red_circle:` emoji in a Slack alert channel.

### 1940: Chain Home and the Birth of "Continuous State"

Back to that green screen.

Radar — RAdio Detection And Ranging — was studied in parallel by
several nations through the 1930s. In Britain, Robert Watson-Watt's
team demonstrated the first effective radar detection in 1935. By the
time the Battle of Britain broke out in 1940, the UK had built the
**Chain Home** network across its southern and eastern coasts.

Chain Home was dozens of steel towers, each broadcasting
megawatt-class radio waves, with a detection radius over 200
kilometers. Reflected signals were turned by PPI operators into
bearing, range, and estimated altitude, and called through to the
operations room's plotting table.

Chain Home did something without precedent: **it turned a dynamic
threat in three-dimensional space into real-time visualization on a
two-dimensional plane**.

This was a phase change in observability.

Beacon towers, telegraph, factory signal lights — all reported
**discrete events**. Chain Home reported **continuous state**. What
the operator saw on the screen was not "the enemy has arrived" (an
event); it was **"the enemy is at this bearing, at this range, on this
heading, at this speed" — a continuously updating state**.

The QPS curve on your Grafana dashboard today is the digital form of
those dots on the Chain Home screen. **Different object. Same
structure.**

---

## Chapter II · The Cold War Raised Monitoring

### 1958: SAGE — The First Computer System Built to Monitor Continuously

The Cold War. America's fear was specific: Soviet Tu-95 bombers
crossing the Arctic over Canada, carrying nuclear warheads bound for
New York, Chicago, and Los Angeles.

From that fear came **SAGE** — Semi-Automatic Ground Environment — a
continental air-defense command system built jointly by the US Air
Force and MIT Lincoln Laboratory. Full operation began in 1958 and
ran until 1983.

SAGE was **the first real-time, large-scale computer-monitoring system
in human history**:

- 23 command centers distributed across the United States
- Each center housed one IBM AN/FSQ-7 computer — **250 tons**,
  occupying half a football field, with 50,000 vacuum tubes per
  paired unit
- Real-time feeds from hundreds of radar stations and thousands of
  communication nodes
- **24×7 continuous operation**. Downtime counted as a national-defense
  incident

SAGE produced a list of firsts:

- First use of a large-scale computer for **continuous status
  monitoring**
- First **graphical operator interface** — officers pointed at the
  screen with a **light pen** to assign targets to intercepting
  fighters. In 1958.
- First system-level **on-call rotation** — operators on eight-hour
  shifts, a human always staring at the screen

If you have ever worked a NOC shift on the back-end of a Datadog SaaS
deployment, what you did was structurally identical to what a 1958
SAGE operator did: **watch the screen, read the metrics, identify
anomalies, escalate**.

Six decades on, the work of the NOC has not fundamentally changed.
The tools have. **The human posture in front of the screen has not.**

### 1969: Unix and the Birth of the Log

1969. Bell Labs.

Ken Thompson and Dennis Ritchie began writing a small operating system
on an idle DEC PDP-7. They called it **Unix**.

Unix brought many things: file systems, pipes, small sharp tools. But
its contribution to observability is a concept so foundational that
no one today remembers it hadn't always existed:

**The system writes its own state out as text, to a place someone
else can read.**

This is the log.

Before Unix, computer state was read mostly from **console lights on
the operator's panel**. Unix made system state **structured plain
text** — which could be `grep`ed, `awk`ed, `sort`ed, redirected to a
file, piped to another program, streamed across the network, audited
after the fact.

**This was the phase change from mechanical to digital
observability.**

Without it, every log-based monitor, alert, and trace you use today
would be impossible.

### 1980: syslog Falls Out of sendmail

Early 1980s. UC Berkeley.

A graduate student named **Eric Allman** was writing a mail server
called **sendmail** (yes — the same sendmail that the entire internet
would then endure for the next thirty years).

sendmail needed to record its own state, but Unix at that time had no
standard logging mechanism — each program wrote its own log file. For
sendmail, Allman wrote a daemon on the side, called **syslog**:

- All programs send log messages to the syslog daemon
- The daemon writes to files by configuration, or **forwards the
  message over UDP to a remote syslog server**
- Each message carries a **facility** (where it came from) and a
  **severity** (how bad it is)

That was the beginning of **log centralization**. A byproduct of a
mail-server project became the default logging mechanism on every
Unix-like system, standardized as RFC 5424 in 2009.

Allman could not have known that the small daemon he wrote to help
debug sendmail would give every sysadmin on earth, for the next forty
years, a professional life defined by `tail -f /var/log/syslog`.

### 1990: SNMP — A Common Tongue for Network Monitoring

May 1990. The internet was just starting to cross from academic to
commercial use.

RFC 1157 was published: **Simple Network Management Protocol
(SNMP)**.

Before SNMP, every network-equipment vendor (router, switch, server)
spoke its own monitoring protocol. To monitor a heterogeneous network
was to install five different management packages.

What SNMP did seemed modest at the time, but its consequences were
deep — **it standardized the act of "reading one metric from one
device"**:

- **OID** (Object Identifier) — a globally unique numeric address for
  every monitorable metric (things like `1.3.6.1.2.1.1.3.0`)
- **GET / SET / TRAP** — three primitive operations
- Over **UDP** — simple, stateless, good enough

SNMP is ugly. Using it is worse. But it won — **because it established
the common tongue**.

This was the observability ecosystem's **first cross-vendor standard**.

Thirty years later, OpenTelemetry does essentially the same thing for
application observability — builds a language every vendor can speak.
**History repeats.**

### 1999: Nagios — The Starting Gun for Open-Source Monitoring

1999. United States. A hobbyist developer named **Ethan Galstad**
worked at home on a side project called **NetSaint**.

It was simple: a Unix daemon that periodically pinged or curled a list
of hosts and ports, and emailed an alert on failure.

In 2002, due to trademark issues, NetSaint was renamed **Nagios** —
Nagios Ain't Gonna Insist On Sainthood, in the acid humor of a
programmer.

Nagios's code wasn't beautiful. Its config syntax bordered on hostile.
Its UI looked like a 1998 BBS. But it got a few things right:

- **Open source** — anyone could modify, extend, build plugins
- **A minimal plugin contract** — a `check_*` executable, return code
  0/1/2/3 for OK / WARN / CRIT / UNKNOWN. That three-line contract
  spawned **thousands of community plugins**
- **Active polling** — rather than waiting for pushes, Nagios came and
  looked

Before Nagios, the open-source world had no serious unified monitoring
entry point. After Nagios, a small company could monitor hundreds of
hosts with a few thousand lines of configuration, **for free**.

For the decade from 2005 to 2015, Nagios ruled the small-and-mid-sized
open-source monitoring market. It was not elegant. But it **opened
the door**.

---

## Chapter III · The Prometheus Moment

### 2003: Borgmon Inside Google

Around 2003. Mountain View.

Google's engineers built an internal monitoring system called
**Borgmon** to observe their container scheduler Borg. Borgmon and
Nagios diverged on a few critical design choices:

- **Time series as the primitive** — every metric is a time series,
  not a boolean check
- **Multi-dimensional labels** — the same metric can carry multiple
  dimensions:
  `http_requests_total{method="GET", status="200", endpoint="/api"}`
- **A query language tailored to time series** — the direct ancestor
  of PromQL

Borgmon was never open-sourced. But Google's **SRE** (Site Reliability
Engineering) culture took shape around Borgmon and, with the 2016
*Site Reliability Engineering* book, broadcast a complete production
monitoring methodology to the rest of the industry.

The term "SRE" itself was coined around 2003 inside Google by **Ben
Treynor Sloss**, who led the shift from "operations" to "software
engineers doing operations."

A decade later, this paradigm would walk out of Google in the shape
of Prometheus.

### 2012: The First Commit in Berlin

2012. SoundCloud's Berlin office.

Two engineers — **Julius Volz** and **Matt Proud** — both Google
alumni, both with Borgmon wired into their reflexes, were looking at
SoundCloud's monitoring stack (StatsD + Graphite + Nagios) and
thinking: for a fast-moving microservice architecture, **this is
nowhere near enough**.

So they started writing a new monitoring system. Inspired by Borgmon.
Named **Prometheus** — the Titan who, in Greek myth, stole fire for
humanity.

Late 2012: the first commit went up on GitHub.

Prometheus brought the community a few things it had been missing:

- **Pull-based** — monitored services expose a `/metrics` endpoint;
  Prometheus periodically scrapes it. This counterintuitive design
  turned out to solve a lot of the consistency problems of push-based
  systems
- **Multi-dimensional label model** — the open-source form of
  Borgmon's core idea
- **PromQL** — not Turing-complete, but deeply engineer-friendly
- **Service discovery** — a natural fit for Kubernetes

In 2015, Prometheus became the **second CNCF project ever graduated**,
after Kubernetes.

It was not the first time-series monitoring system. Not the fastest.
Not the most feature-complete.

But it **appeared in the right place at the right time** — exactly as
Kubernetes was taking off. Ten years on, Prometheus is the de facto
standard for cloud-native monitoring.

### 2014: Grafana — Making Monitoring Beautiful

2014. Stockholm. **Torkel Ödegaard** sat at his own computer and
forked Kibana 3.

He was working at Orbitz, and the team was using Graphite. He didn't
like Graphite's built-in UI. He didn't like any monitoring UI of the
era. He wanted one that looked **good**.

A few months later, **Grafana** shipped. Open source.

Grafana didn't do anything revolutionary — it was just a front end.
But it got one thing right: **it made monitoring dashboards into
things that could be treated as design**.

Dark backgrounds. Soft palettes. Clean typography. Responsive
layouts. Draggable panels. Everything now taken for granted in a
dashboard was largely absent in monitoring tools before Grafana.
Nagios looked like a BBS. Graphite's UI looked like a 1999 student
project.

Grafana changed an entire generation of engineers' **expectation of
what monitoring should look like**.

By 2024, Grafana supports **more than 150 data sources** and is the
de facto visualization layer of the observability stack. Torkel's
Grafana Labs was valued at **three billion dollars** in 2022.

One person. At home. Forking an open-source project. Ten years later,
that fork had changed an industry's sense of taste.

### 2019: OpenTelemetry — A Common Tongue, Take Two

1990: SNMP built a common tongue for network monitoring. 2019:
application observability's turn.

Under CNCF's stewardship, two projects merged:

- **OpenTracing** (2016) — an open standard for distributed tracing,
  championed by names like LightStep's Ben Sigelman
- **OpenCensus** (2017) — an open standard for metrics and tracing,
  championed by Google

The merged project: **OpenTelemetry** (OTel). The goal: **one set of
observability-data standards spoken by every vendor, every language,
every tool**.

OTel is more ambitious than SNMP. It aims to unify **metrics, traces,
and logs** — not just metrics. It defines semantic conventions —
how an HTTP span should be attributed, how a database call should be
tagged.

The project is still evolving fast. It isn't perfect. Many SDKs are
still stabilizing. The semantic-conventions spec changes monthly.
None of that is what matters.

What matters is that **for the first time in history, application
observability has a common tongue the way network monitoring has had
since SNMP**.

---

## Closing Out Book II: A Hundred and Eighty Years Later

From Morse's first copper wire in 1837 to the first OTel spec in
2019 — **one hundred and eighty years**.

In that time, humanity swapped the tools for "seeing the distance"
from iron and copper to electricity and silicon.

- Speed: up by **10^9**
- Data volume: up by **10^15**
- Team scale: from a national engineering project of hundreds (SAGE)
  to one person forking a project at home and changing an industry
  (Grafana)

But:

**The beacon tower's need hasn't moved** — Nagios still asks: *is
the host alive?*

**The observatory's need hasn't moved** — Prometheus still
continuously samples system metrics.

**The court chronicle's need hasn't moved** — syslog still commits
every system event to an immutable record.

**Nightingale's need hasn't moved** — Grafana still turns data into
charts that can move an organization.

Three-thousand-year-old needs, met with twenty-first-century tools.
The tools have multiplied thousands of times. **The needs themselves
have not moved.**

---

## 🜚 Next · Book III · *The Divergence and the Echo*

When Prometheus was born in Berlin in 2012, nobody could have
predicted that in the next ten years, around this one stack, East and
West would grow **two utterly different ecosystems**.

If you've read Book I, that should sound familiar.

**Book III** maps the modern divergence. Why didn't PagerDuty make
it into China? Why did DeepFlow take up the position Pixie left
behind? Why couldn't Datadog's business model survive domestically?
Why has China once again grown a **"Qintianjian-style"**
observability system instead of a **"Nightingale-style"** one?

And — why was this divergence not actually born in 2012. It was
already **written three thousand years ago**.

The author of this project will, at the end of Book III, explain why
he built something called `tsdb-operator`.

Stay with me.
