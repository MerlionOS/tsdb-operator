# The Observability Epic · Book I: The Flame and the Eyes

> 中文: [wechat-epic-1-flame-and-eyes.zh.md](./wechat-epic-1-flame-and-eyes.zh.md)

> From a single plume of smoke atop Mount Li in 771 BCE,
> to a rose diagram drawn from a Crimean battlefield in 1858 —
> this is two thousand years of the human race trying,
> by every possible means, **to see into the distance**.

---

## Prologue: A Smile, and the End of a Dynasty

771 BCE. A woman named Bao Si laughed.

She didn't laugh easily. Concubine to King You of Zhou, she was cold by
nature, and rarely showed expression. The king, desperate to win a smile
from her, came up with an idea: **light the beacon fires on Mount Li**.

The flames went up. The lords of the realm, seeing smoke columns rise
from the royal beacons, assumed invaders had breached the border and
raced through the night to answer the call. They arrived at Mount Li,
armored and armed, panting and sweat-soaked — only to find King You at
his tower, drinking wine, a woman at his side covering her face in
laughter.

Bao Si laughed. The king had won her smile for a moment.

The price came three years later. The Quanrong did invade. The beacons
blazed again. **No lord answered.**

The Western Zhou dynasty fell.

This is the most famous false alert in Chinese history. Twenty-seven
centuries later, when a modern engineer taps "mute this alert" in a
Slack channel, they are following the same logic:

**When signals are too frequent and carry no weight, signals die.**

Observability was not invented in Silicon Valley.

It is one of humanity's oldest needs: **in the places you cannot see,
before something goes wrong, let someone tell you**.

This is an epic about observability, spanning from three thousand years
ago to today. Book I covers the ancient world: how China built a
**state-level, centralized observation system** with beacon towers, the
Imperial Astronomical Observatory, the post-relay network, and the
court chronicles; how the West took a different path, through the myth
of the hundred-eyed giant, the Roman census, a patriot's lanterns, and
a nurse's charts.

Both paths still shape the monitoring system on your screen right now.

---

## Chapter I · East: A Monitoring Center for an Empire

### The Beacon Towers: A Multi-Region Alerting System, Three Thousand Years Old

Beacon towers, at their core, were a **multi-hop alerting relay**.

Standard layout: a tower every 5 to 10 Chinese *li*, manned by year-round
watchmen. When the next tower in the chain lit up, yours lit up too.
Wolf dung burned by day — the smoke was thick enough to be seen in full
sun. Dry wood burned by night — the flames carried tens of kilometers
through the dark. A signal from the frontier could reach Chang'an in a
few hours.

Translated into engineering terms:

- **Topology**: multi-hop mesh, each node forwarding to the next
- **Alert channels**: smoke by day, fire by night — different media for
  different conditions
- **SLA**: a fast horse does 300 li a day; beacon fire does 1,000 li an
  hour — a three-order-of-magnitude improvement
- **Cost of a false alert**: the lords and the beacon fire → real alerts
  ignored → the collapse of a kingdom

This is an alerting pipeline implemented in pure physical-layer
hardware. No electronics. No automation. **Yet every core concept of
modern alerting is already there.**

There was more. Beacon towers didn't transmit just one bit of "enemy
incoming." The Tang-era *Wei Gong Bing Fa* records that watchmen would
light one, two, three, or four piles of fire depending on the invading
force — from skirmish, to medium raid, to major invasion, to the main
army itself.

That's 2 bits of state, possibly more. The modern equivalent is shipping
a severity level plus an impact tag in your alert payload.

Three thousand years ago, a frontier lieutenant was already thinking
about the alert-tiering table in your PagerDuty setup.

### The Imperial Astronomical Observatory: A State-Level SRE Team

From the Qin and Han dynasties onward, every Chinese dynasty ran a
dedicated institution for astronomical observation. Han called it the
Taishi Ling; Tang, the Sitiantai; Yuan, the Taishi Yuan; Ming and Qing,
the **Qintianjian** (Imperial Astronomical Observatory). Names changed.
The function did not — **a full-time, 24-hour, nationwide observation
service**.

Responsibilities:

- **Shift-based watch**: day, night, dawn, and dusk. Night-sky observers
  could not close their eyes
- **Continuous metrics**: solar and lunar eclipses, comets, meteors,
  planets, cloud formations, earthquakes
- **Structured output**: almanacs, star charts, memorials — every
  document in a fixed format
- **SLO**: a wrong observation got you caned. A wrong eclipse prediction
  got you beheaded

No exaggeration. The Yuan-dynasty astronomer Guo Shoujing, in his
*Shoushi Calendar* of 1281, measured the length of a solar year as
**365.2425 days**. The modern measurement differs by **26 seconds**.

What does this mean? It means that behind Guo's number was **generations
of disciplined data-quality control**, an empire-scale observation team
running for centuries without mechanical clocks, without telescopes,
purely with the naked eye and bronze-iron instruments.

The armillary sphere, the simplified armilla, the sundial, the
clepsydra, the gnomon — a full observation tool chain. Each piece
pushed the limits of what was engineerable at the time.

The curves, dials, and heatmaps on your Grafana dashboard today are
structurally no different from those bronze instruments on the
observatory roof: **they take invisible changes and render them into
visible markings**.

### The Imperial Postal Relay: A Message Queue That Ran for Two Thousand Years

Starting in the Han, China ran a full postal-relay system.

- "One *pu* every 10 li, one *yi* every 30 li" — standardized topology
- "Eight-hundred-li urgent" — independent priority channel for emergency
  messages
- When a courier's horse died mid-route, the next station took over —
  **built-in retry**
- Separate land and water routes, northern and southern tracks —
  **multi-path redundancy**
- Every station logged the documents passing through — **trace ID +
  span log**

This is a distributed messaging system that ran for **two thousand
years** and covered an entire empire.

Every problem you handle with Kafka today — latency, message loss,
ordering, priority, path selection, failover, audit trail — was thought
through, and thought through again, by the Han dynasty's Minister of
Posts. Every imperial expansion, every war, every disaster relief was a
load test on this system.

Under Kangxi, during the Revolt of the Three Feudatories, an urgent
military dispatch from Yunnan to Beijing took **nine days**. Slow by
modern standards, but the distance was **2,500 kilometers** over
**eighty relay stations**, with zero electronic equipment — an average
per-hop latency of a few hours.

Put that figure next to the replication lag of a modern Kafka cluster,
and your feelings get complicated.

### The Court Chronicle: A Three-Thousand-Year Audit Log

From the Zhou onward, China ran an institution called the **Qijuzhu** —
the Court Chronicle.

The setup: a dedicated historian followed the emperor everywhere,
recording every word he spoke, every decision he made, every person he
received, every memorial he ruled on.

There was one iron rule: **the emperor could not read his own
chronicle**.

- **Immutable by design**: the emperor had no authority to edit
- **Retention**: measured in millennia — surviving fragments from Han
  and Tang chronicles are still read today
- **Compliance basis**: the *Rites of Zhou* codified historian
  independence
- **Downstream use**: primary source material for dynastic histories

Lay these design principles next to modern SOC 2 audit requirements —
**write-once, separation of duties, third-party verifiability,
long-term retention** — and you find almost the same specification.

Three thousand years ago, the Confucian court had already worked out
"the operator cannot delete their own operation log." And they worked
it out not because PCI-DSS existed, but because they understood
something deeper: **the value of observation depends on the
observation itself being uncorrupted by the observed**.

### The East, Summed Up

Ancient Chinese observability has one sharp feature: **observation is
an act of the state**.

The beacons were military. The Qintianjian was imperial. The postal
relay was bureaucratic. The court chronicle was the dynasty's. Every
byte of observational data they produced flowed toward one destination:
**centralized decision-making**.

This gene still runs through modern Chinese observability products,
which tilt toward management visibility — multi-tenancy, hierarchical
permissions, compliance audit, centralized reporting. That's not
coincidence.

Design philosophy: **centers first, then distributed nodes**.

---

## Chapter II · West: Eyes, Numbers, Lanterns, and a Nurse

### Argus Panoptes: Observation As Power

Greek myth has a giant named Argus Panoptes.

*Panoptes* in ancient Greek meant "all-seeing."

Argus had a hundred eyes covering his body. When he slept, some always
stayed open. The goddess Hera set him to guard Io — her rival, a mortal
woman Zeus had turned into a cow.

Hermes came to rescue Io. He played his flute until every one of Argus's
hundred eyes finally closed, then beheaded the giant in a single stroke.
Hera, grieving, placed Argus's hundred eyes on the tail of the peacock.

The myth left a root: **panoptic** — all-seeing.

Two thousand years later, the English philosopher Jeremy Bentham
designed a prison structure called the **Panopticon** (1791) — a
central watchtower surrounded by cells open to its view, where the
prisoners could never know whether a guard was watching. They had to
**assume they were always being watched**.

The French philosopher Michel Foucault, in *Discipline and Punish*
(1975), extended the Panopticon into a **metaphor for the essence of
power**: the observer's position creates authority; the observed begin
to regulate themselves.

From myth to philosophy, the Western line tells one story:

**"All-seeing" has been humanity's unchanging obsession for two
thousand years.**

Today's full-stack eBPF observability, distributed tracing, full-stack
APM — all of it still chases the ghost of Argus Panoptes:
**put every occurrence on earth inside someone's eyes**.

This is a distinctly Western motif. The Chinese placed observation with
the state. The West turned observation into a **philosophical problem**:
**who is watching whom? where is the observer? how does observation
itself change the world?**

### The Roman Census: A Five-Year Batch ETL

Around 6 BCE, the emperor Augustus issued an edict: **every subject of
the empire was to register**.

This was the Roman **census**.

- **Purpose**: taxation plus conscription
- **Cadence**: every five years
- **Coverage**: every Mediterranean province
- **Output**: family-level records, archived in official state records

A textbook batch data pipeline: periodic full scans, structured
enrollment, centralized storage, downstream consumption.

According to the Gospel of Luke, Jesus's parents Joseph and Mary walked
from Nazareth to Bethlehem in order to register for this census. Which
makes it, in a strict sense, **the most famous ETL job ever run — one
whose output was the birth narrative of Christianity**.

The Romans went a different way from the Chinese. The Chinese did
continuous observation (the Qintianjian staring up every day). The
Romans did periodic full scans (the census every five years).

This batch-ETL gene persisted all the way to the corporate data
warehouse of the twentieth century:
**scheduled jobs + rigid schema + service to decision-making**.

Your Airflow DAGs, your dbt models, your Snowflake tasks — all
descend, in spirit, from Augustus's census.

### Paul Revere's Lanterns: Binary Alerting, 1775

The eve of the American Revolutionary War.

April 18, 1775. Boston. Robert Newman, the sexton of the Old North
Church, had an arrangement with a silversmith named **Paul Revere**.

**"One if by land, two if by sea."**

If the British marched by land, one lantern in the belfry. By water,
two.

Unpack the engineering:

- **Encoding**: 1 = land, 2 = sea (2 bits, with room for error
  detection)
- **Signal medium**: lanterns — visual, visible at night, readable at
  distance
- **Threshold**: any lantern triggers
- **Downstream action**: Revere mounts, rides toward Lexington, pounds
  on every militia door along the way — *"The British are coming!"*

By dawn, the militia at Lexington Green was assembled and armed. The
British advance guard walked into them. The first shot of that morning
was later called **"the shot heard round the world"** — the American
Revolution had begun.

The whole alerting system ran for less than a minute. But it turned one
bit of signal into a **tactical first-mover advantage in a war**.

What Revere did, in modern terms: **translate signal into executable
decision**. That's what PagerDuty claims to do today. An 18th-century
silversmith got there first.

### Florence Nightingale: The Spiritual Grandmother of the Modern Dashboard

1854. The Crimean War. British troops were dying in extraordinary numbers
on the north coast of the Black Sea. Domestic outrage was mounting. A
nurse named **Florence Nightingale** was deployed to the British field
hospital at Scutari.

She was not a general. Not an official. Not a doctor. But she had a
habit that was radical for its time: **she recorded everything**.

How many died each day. From what cause. Gunshot wounds? Broken bones?
Dysentery? Typhus? Pneumonia? Frostbite? She ordered her subordinates
to tabulate deaths by cause, day by day.

After the war, in 1858, she turned the numbers into a report for the
British government. The report contained a chart later called the
**Rose Diagram** or **Coxcomb Chart**: twelve months arranged in a
circle, each month's death toll expressed as the area of a wedge,
color-coded by cause.

The conclusion was **impossible to refute**:

**British soldiers died of preventable sanitary disease in vastly
greater numbers than they died of combat.**

The British government overhauled the army's medical system on the
strength of that chart. In 1859 Nightingale was elected **the first
female member** of the Royal Statistical Society.

Her chart was **the first use of visualization in history to drive a
major organizational decision**.

Nightingale was not a philosopher. Not a general. Not an officially
appointed observer. She was a nurse changing bandages at the front.
Yet through **her own collection, her own analysis, her own
visualization**, she moved an empire to reform itself.

The Grafana dashboards you build, the SLO reports you publish, the
incident postmortems you draw — all of them chase Nightingale's 1858
rose.

**She is the spiritual grandmother of the modern dashboard.**

### The West, Summed Up

Ancient Western observability never built anything like China's
official observational bureaucracy. But it grew something else:
**the authority of the individual observer**.

Bentham was a philosopher. Paul Revere was a silversmith. Nightingale
was a nurse. None of them were "officially appointed to observe" — yet
through **their own observation and record-keeping, they bent the arc
of history**.

This line continues to the present. Silicon Valley observability is
**born from the engineering community** — Prometheus was a side project
by a handful of SoundCloud engineers; Grafana started as Torkel
Ödegaard's personal fork; OpenTelemetry grew bottom-up from developers.

**East: centralized, top-down observation.**
**West: decentralized, bottom-up observation.**

Three thousand years later, both paths are still clearly visible.

---

## Closing Out Book I: Two Paths, One Anxiety

The beacon fire that made Bao Si laugh, and the lanterns Paul Revere
had hung in a church belfry — twenty-five hundred years apart, half a
planet between them, and yet they are doing the same thing:
**turning distant danger into a local signal**.

The armillary sphere on the Imperial Observatory roof, and the rose
diagram drawn by a British nurse — one cast in bronze, the other drawn
in ink, and yet they are also doing the same thing:
**turning hard-to-read complexity into data a decision-maker can act
on**.

The court chronicle and the Panopticon are a deeper mirror of each
other: **the mere presence of the observer changes the behavior of the
observed**. That insight was arrived at independently — in the
Confucian court and in the English study — across three millennia and
two civilizations.

You open your monitoring dashboard today. You watch a line jitter up
and down. You think you are using 2020s-era engineering.

**You are actually inheriting a tradition that stretches, unbroken,
back to the Zhou dynasty.**

The tools have changed for three thousand years. **The anxiety of
information asymmetry has not moved.**

---

## 🜚 Next · Book II · *Iron and Lightning*

With the Industrial Revolution, observability meets its real opponent —
**speed**.

When the telegraph travels thousands of kilometers in a second, when
radar scans skies the eye cannot see, when computers generate more logs
than humans can read — observability faces, for the first time, a
problem the ancients never had:

**the data itself is too large to be observed**.

Book II tells that story — from Samuel Morse's first telegraph key in
1837, to a Berlin engineer named Julius Volz pushing the first commit
of Prometheus to GitHub in 2012. A hundred and eighty years in which
humanity swapped the tools for "seeing the distance" from **iron and
copper, to electricity and silicon**.

Stay with me.
