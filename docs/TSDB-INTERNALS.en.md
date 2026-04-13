# How Prometheus TSDB Works: Internals That Informed tsdb-operator's Design

中文版: [TSDB-INTERNALS.zh.md](TSDB-INTERNALS.zh.md)

Written for readers who know Prometheus at the operational level and want
to see why the primitives the operator exposes (block duration flags,
admin-API snapshot, `/-/ready` probe, resource limits) are the natural
choices given how the TSDB actually works.

Every section ends with a **Design Impact** note pointing at the specific
file in this repo that cashes the concept.

## 1. TSDB Overview

A Prometheus TSDB instance runs two layers:

- **Head block** — the in-memory (plus mmap'd tail) write path. Every
  scraped sample lands here first. Latest ~2 hours of data.
- **Persistent blocks** — immutable on-disk directories. Each contains
  chunks, an inverted index, a metadata file, and a tombstone log.

```
scrape ──▶ append ──▶ WAL (fsync) ──▶ Head (memSeries + chunks)
                                       │
                                       ▼ every 2h
                                 compaction cuts a block
                                       │
                                       ▼
                              /prometheus/<ULID>/
                                 ├── chunks/000001
                                 ├── index
                                 ├── meta.json
                                 └── tombstones
```

Reference: [`prometheus/tsdb/db.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/db.go),
[`prometheus/tsdb/head.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/head.go).

**Design Impact.** Raw file-copy of `/prometheus/` is unsafe: the head
writes constantly and files are renamed during compaction. The right
primitive is Prometheus's own snapshot API, which creates a hard-linked,
consistent view of the completed blocks plus (optionally) a head
snapshot. That's why
[`internal/backup/backup.go:155`](../internal/backup/backup.go) POSTs to
`/api/v1/admin/tsdb/snapshot` before we tar anything off disk.

---

## 2. WAL (Write-Ahead Log)

The WAL makes ingestion durable across process restarts. Each `Append`
writes a record before the sample is visible to queries. Record types:

- **Series** — a new `(labels) → series ID` mapping
- **Samples** — `(series ID, timestamp, value)` tuples
- **Tombstones** — deletion intent (see §4)

Segments rotate at ~128 MiB. On boot the head block is rebuilt by
replaying the WAL (plus checkpoints, which are compacted WAL segments).

There is a **second, separate WAL** for `remote_write`: it buffers
samples while the remote endpoint is unreachable. This is a distinct
shard-queue subsystem, not the ingestion WAL. Reference:
[`prometheus/storage/remote/wal_watcher.go`](https://github.com/prometheus/prometheus/blob/main/storage/remote/wal_watcher.go).

**Design Impact.** The WAL is inherently per-instance; there is no
officially supported way to share it across replicas. This is why
`tsdb-operator` does **not** attempt to build a custom WAL-sharing HA
story. Instead:

- Run `spec.replicas >= 2` — each Prometheus has its own WAL, each one
  is durable on its own PVC.
- Deduplicate at query time with Thanos Query, or push to a durable
  backend with `spec.remoteWrite` (see
  [`api/v1/prometheuscluster_types.go`](../api/v1/prometheuscluster_types.go)
  `RemoteWriteSpec`).

The `remote_write` WAL buffer is precisely what makes `remoteWrite` a
viable HA integration point: short receiver outages don't lose data.

---

## 3. Chunk Storage & Gorilla Compression

Samples are grouped into **chunks** of up to 120 samples (~2 h at a 1 m
scrape interval). Each chunk uses Gorilla encoding
([Facebook's 2015 VLDB paper](https://www.vldb.org/pvldb/vol8/p1816-teller.pdf)):

- **Timestamps**: delta-of-delta — store the change in inter-sample
  interval, usually zero for steady scraping.
- **Values**: XOR with the previous value; encode only the run of
  changed bits.

Typical compression: ~1.37 bytes per sample after encoding.

```
Chunk lifecycle:
  open (head) ──▶ full (120 samples or 2h) ──▶ mmap'd on disk
                                               ──▶ merged into a block
```

**Design Impact.** When `spec.thanos.enabled` is true we pin
`--storage.tsdb.min-block-duration=2h` **and** `max-block-duration=2h`
([`internal/controller/prometheuscluster_controller.go:271`](../internal/controller/prometheuscluster_controller.go)).
Two reasons:

1. Thanos sidecar ships 2 h blocks; pinning Prometheus to the same cadence
   keeps shipping behaviour predictable.
2. Thanos does its own downstream compaction on object storage — having
   Prometheus compact locally first would produce blocks Thanos can't
   upload cleanly.

Retention (`spec.retention`, default 15 d) is a separate knob and is
**not** touched by the Thanos opt-in.

---

## 4. Compaction

Compaction merges small blocks into larger ones and permanently removes
tombstoned series. Roughly:

- Head cuts a 2 h block.
- Planner merges adjacent blocks once their combined range fits the
  next level.
- Default ratios: each level's max duration scales with retention
  (`max_block_duration = min(retention/10, 31d)`). A 15 d retention
  caps at 1.5 d blocks; a 1 y retention caps at 31 d.
  See [`tsdb/compact.go`](https://github.com/prometheus/prometheus/blob/main/tsdb/compact.go).

Tombstones are lazy: `DELETE` via the admin API writes a tombstone record;
the series is only physically removed at the next compaction that
includes that block.

**Design Impact.** The snapshot API returns **hard links** into the
current blocks plus the head. Hard links share inodes, so even if
compaction rewrites the block while we're uploading, our tar is reading
from the linked directory that doesn't get rewritten — the filesystem
keeps it alive until the last reference drops. That's why we don't need
to coordinate snapshot timing with compaction state in
[`internal/backup/backup.go`](../internal/backup/backup.go): the hard-link
semantics make it safe to run the backup cron on wall-clock time. The
trade-off is that we may back up a snapshot that lacks the just-cut
block; the next cron run picks it up.

---

## 5. Index

Each persistent block has an inverted index:

```
label name ──▶ label value ──▶ [series IDs] ──▶ [chunk refs]
  __name__       up              [3, 17, 42]     [(0x...), (0x...)]
  job            prometheus      [3, 17]         ...
```

PromQL selectors (`{job="prometheus",instance=~"…"}`) are translated into
posting list intersections. Cardinality — the **number of distinct label
combinations** — directly determines the size of the postings lists in
memory.

> High cardinality is the #1 cause of Prometheus OOM in production.
> Common culprits: labels derived from user IDs, request IDs, trace IDs,
> raw URL paths.

**Design Impact.** `spec.resources` on `PrometheusCluster` is a
first-class field — see
[`api/v1/prometheuscluster_types.go`](../api/v1/prometheuscluster_types.go)
`PrometheusClusterSpec.Resources`. Platforms that manage fleets of
Prometheus via this operator should:

- Set memory *limits* conservatively; an OOM kill recovers cleanly via
  WAL replay.
- Pair with VPA / HPA on memory utilisation rather than on CPU.
- Alert on `prometheus_tsdb_head_series` growth rate, not absolute count.

---

## 6. Head Block

The head block is a `memSeries` table plus the currently-open chunks.
Since Prometheus 2.19, head chunks are **mmap'd**: the file on disk is
the source of truth, not the in-memory buffer. A crash loses at most the
current partial chunk plus anything post-last-fsync in the WAL.

When the head's time range exceeds `chunkRange` (default 2 h),
compaction cuts the in-range series into a block and truncates the
head.

**Design Impact.**
[`internal/ha/ha.go:81`](../internal/ha/ha.go) polls `/-/ready`, not
`/-/healthy`:

- `/-/healthy` → process is alive.
- `/-/ready` → head block has finished WAL replay; TSDB can serve
  queries.

Freshly-started Prometheus can be healthy for many seconds before it's
ready (WAL replay scales with sample count). Probing `/-/ready` is the
right signal for "this replica should receive query traffic". Dual-
probing both endpoints is a reasonable future refinement — it would
distinguish "dead process, reschedule me" from "alive but still
replaying, wait".

---

## 7. Snapshot API

`POST /api/v1/admin/tsdb/snapshot` creates
`/prometheus/snapshots/<timestamp>-<hash>/`:

- For each completed block: a hard-link tree to the original block
  directory.
- By default the current head is also written as a new block (skip with
  `?skip_head=true`).
- Returns `{"status":"success","data":{"name":"<dirname>"}}`.

Consistency: hard links are atomic and reference immutable (for blocks)
or frozen (for head) content. No file corruption risk, no coordination
with compaction needed.

Requires `--web.enable-admin-api`. The operator gates that flag on
`spec.backup.enabled`
([`internal/controller/prometheuscluster_controller.go:262`](../internal/controller/prometheuscluster_controller.go))
— unnecessary admin surface stays closed otherwise.

Our backup flow, end to end:

```
cron tick
  └─▶ POST /api/v1/admin/tsdb/snapshot          (backup.go:155)
  └─▶ parse "data.name" from response            (snapshotResp)
  └─▶ SPDY exec `tar -C /prometheus/snapshots \
        -cf - <name>` on <cluster>-0            (exec.go)
  └─▶ pipe stdout into s3 manager.Upload        (s3client.go)
  └─▶ SPDY exec `rm -rf …/<name>`               (cleanup)
  └─▶ Status().Update(lastBackupTime = now)
```

**Design Impact.** Alternatives were considered and rejected:

- **VolumeSnapshot (CSI)**: couples backup integrity to CSI behaviour,
  inconsistent across cloud providers, and still needs off-cluster
  shipping to meet a "region goes down" bar. See
  [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md).
- **rsync / raw file copy**: race with compaction — files are renamed
  and created mid-flight.
- **Thanos sidecar only**: valid, but forces users into the full Thanos
  stack just to have backups. We keep Thanos as an opt-in.

The backup cron writes the S3 object and stamps
`status.lastBackupTime`; REST-triggered manual backups additionally land
in the PostgreSQL audit log via
[`pkg/api/server.go`](../pkg/api/server.go)'s
`triggerBackup` handler (cron-path backups do not currently go through
the audit logger).

---

## 8. Summary — TSDB concepts, cashed in this repo

| TSDB concept | Where it shows up in tsdb-operator |
|---|---|
| WAL is per-instance, non-shareable | Run N replicas; dedup via Thanos Query or `spec.remoteWrite`; don't invent custom WAL sharing. |
| Snapshot API = hard-link consistency | `backup.go` POSTs the API, tars the snapshot dir, uploads via multipart. No compaction coordination needed. |
| Head chunks are mmap'd; `/-/ready` means "done replaying WAL" | `ha.go` probes `/-/ready`, not `/-/healthy`. |
| Block duration independent of retention | Thanos opt-in pins `min/max-block-duration=2h`; `spec.retention` is orthogonal and unchanged. |
| Cardinality drives memory | `spec.resources` is first-class; docs warn operators to monitor series growth. |
| Tombstones are lazy | Deleted data isn't physically gone until compaction. Callers relying on GDPR-style deletion need to trigger a manual compaction. |
| `--web.enable-admin-api` is load-bearing for snapshots | Controller appends the flag only when `spec.backup.enabled` is true. |

## Further Reading

- Fabian Reinartz, [Writing a Time Series Database from Scratch](https://fabxc.org/tsdb/) — the original TSDB design essay.
- Pelkonen et al., [Gorilla: A Fast, Scalable, In-Memory Time Series Database](https://www.vldb.org/pvldb/vol8/p1816-teller.pdf) — the compression paper this TSDB is descended from.
- [Prometheus TSDB format spec](https://github.com/prometheus/prometheus/blob/main/tsdb/docs/format/README.md) — block directory layout, index format.
- [Thanos block format](https://thanos.io/tip/thanos/storage.md) — what the sidecar ships.
- [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md) — why we picked scheduled snapshots over continuous `remote_write` as the default.
