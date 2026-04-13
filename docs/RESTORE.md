# Restore runbook

中文版: [RESTORE.zh.md](RESTORE.zh.md)

How to bring a Prometheus cluster back from a backup that `tsdb-operator`
shipped to S3/MinIO.

> **Backup model.** Since v0.6.0 the scheduler tar-streams the on-disk
> snapshot directory (`/prometheus/snapshots/<ts>/`) from the Prometheus
> pod into S3 via multipart upload, then deletes the directory to free
> disk. The archive contains real TSDB blocks (chunks, index, meta.json)
> and is restorable. See
> [ADR-0002](adr/0002-scheduled-snapshots-vs-continuous-remote-write.md)
> for the rationale.

## When to restore

- PVC deleted or corrupted.
- Region/AZ lost, rebuilding in another cluster.
- Drill — periodic proof that your backups are usable.

## Prerequisites

- `tsdb-ctl` on your workstation:
  `go install github.com/MerlionOS/tsdb-operator/cmd/tsdb-ctl@latest`
- Credentials to the backup bucket exported as `AWS_ACCESS_KEY_ID` and
  `AWS_SECRET_ACCESS_KEY`.
- A `PrometheusCluster` CR (can be empty — we need the StatefulSet/PVC to
  restore *into*).

## Step 1 — List available snapshots

```bash
tsdb-ctl list \
  --bucket tsdb-backups \
  --prefix demo \
  --endpoint http://minio.example.com   # omit for real AWS S3
```

Output is newest-first:

```
2026-04-13T00:19:00Z          72  demo/demo/20260413T001900Z.tar
2026-04-13T00:18:00Z          72  demo/demo/20260413T001800Z.tar
```

## Step 2 — Download a snapshot

Either take the newest automatically:

```bash
tsdb-ctl restore \
  --bucket tsdb-backups --prefix demo \
  --endpoint http://minio.example.com \
  --dest ./restore
```

Or pin a specific key:

```bash
tsdb-ctl restore \
  --bucket tsdb-backups \
  --key demo/demo/20260413T001800Z.tar \
  --dest ./restore
```

## Step 3 — Stage the archive into the target pod

Pick a replica to restore into. If the cluster is being rebuilt, create
the `PrometheusCluster` first and wait for the StatefulSet to schedule
replica 0.

```bash
# Copy the archive into the pod
kubectl cp ./restore/20260413T001800Z.tar \
  <ns>/<cluster>-0:/prometheus/

# Extract into the TSDB path
kubectl exec -n <ns> <cluster>-0 -- \
  tar -xf /prometheus/20260413T001800Z.tar -C /prometheus/
```

## Step 4 — Restart the replica

```bash
kubectl delete pod -n <ns> <cluster>-0
```

The StatefulSet reschedules, Prometheus loads the restored TSDB blocks on
startup. Confirm:

```bash
kubectl -n <ns> port-forward pod/<cluster>-0 9090:9090
curl 'http://localhost:9090/api/v1/query?query=up' | jq '.data.result | length'
```

## Step 5 — Clean up

```bash
rm -rf ./restore
```

## When the restore fails

- **`ListObjectsV2: NoSuchBucket`** — check `--bucket` and region/endpoint.
- **`tar: unexpected EOF`** — the backup pipeline wrote the admin-API
  marker, not an archive. Adjust Step 3 for your actual artifact shape,
  or fix the backup side first (see the reminder at the top).
- **Pod CrashLoops after restart** — TSDB corruption. Pin an older
  snapshot with `--key` and retry.
- **Metrics are there but the range looks truncated** — Prometheus only
  loads complete blocks. Check `/prometheus/wal/` is empty on the
  restored pod; stale WAL can mask restored blocks.
