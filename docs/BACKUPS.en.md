# Why snapshot Prometheus TSDB to S3?

中文版: [BACKUPS.zh.md](BACKUPS.zh.md)

A PVC on its own is not a backup. Below is the rationale `tsdb-operator` bakes
scheduled S3 snapshots into the `PrometheusCluster` CRD as a first-class feature.

## Why a PVC is not enough

**1. The failure domain of a PVC is small.**
Most StorageClasses (EBS, GCE-PD, local-path) pin a PV to a **single
availability zone**. Lose the AZ, lose the data. Node-disk failures or an
accidentally deleted PVC have the same outcome — PVC replicas are not a
cross-region backup.

**2. Prometheus retention is a bounded local window.**
`--storage.tsdb.retention.time=15d` means anything older is gone. Compliance
regimes often require **≥1 year** of raw data; you will not fit that on block
storage at a reasonable cost.

**3. Block storage is expensive; object storage is cheap.**
EBS gp3 runs ~$0.08/GB/month; S3 Standard ~$0.023/GB/month; Glacier cheaper
still. Keeping only **hot data on PVC** and **cold data in S3** is a big cost
win.

**4. Disaster recovery and human error.**
Accidentally deleted CR, retention knocked down, ransomware, region outage —
you need a copy **decoupled from the cluster** that any new cluster can restore
from.

## How the snapshot actually works

Prometheus exposes an admin API:

```
POST /api/v1/admin/tsdb/snapshot
```

It creates **hard links** to the current TSDB blocks under
`/prometheus/snapshots/<timestamp>/`. This is instantaneous, uses no extra disk
until blocks are rewritten, and does not block ingestion.

`tsdb-operator`'s backup controller then:

```
cron tick
  → POST /api/v1/admin/tsdb/snapshot
  → tar the snapshot directory
  → PutObject to S3 / MinIO
  → update status.lastBackupTime
```

## Difference vs Thanos

Thanos has a different model: a **sidecar continuously ships 2h blocks to
object storage**, so historical data already lives in S3 — there is no distinct
"backup" action. In return you run and maintain Thanos Sidecar, Store Gateway,
and Compactor alongside each Prometheus.

`tsdb-operator`'s stance: if you want to stay on **vanilla Prometheus** without
pulling in the full Thanos stack, scheduled snapshots are the lightest way to
get "there is a copy off-cluster that we can restore from."

## When snapshots are not the right answer

- You need **infinite retention with PromQL** over all of it → use Thanos or
  VictoriaMetrics.
- You need a **global query view** across clusters → use Thanos Query or VM.
- You need **sub-minute RPO** → stream with `remote_write` to a durable
  backend; don't rely on cron snapshots.
