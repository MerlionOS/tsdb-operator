# ADR 0002: Scheduled snapshots vs continuous remote_write for backup

- Status: Accepted
- Date: 2026-04-12

## Context

"Backup" for a Prometheus cluster can mean two very different things:

1. **Continuous replication** — stream every sample to a durable backend via
   `remote_write` (Thanos Receiver, VictoriaMetrics, Grafana Mimir). The
   backend *is* the source of truth.
2. **Periodic point-in-time snapshots** — use Prometheus's admin snapshot API
   to take consistent hard-linked copies of the TSDB and ship them to object
   storage.

Both are valid. We had to pick the default shape of the `spec.backup` feature.

## Decision

Make **scheduled S3 snapshots** the first-class backup mechanism in
`tsdb-operator`. Add `spec.remoteWrite` as an *additional* optional feature
(ADR shipped separately), not a replacement.

Rationale:

- The target audience is teams that want to **keep vanilla Prometheus** and
  avoid pulling in a new durable backend just to have backups.
- Snapshots require zero additional components: a cron entry, the admin API,
  an S3 bucket. Nothing else needs to be operated.
- Disaster recovery, "I accidentally `kubectl delete`d the PVC", and
  compliance (here is the state of the world at time T) are all covered.
- Continuous `remote_write` is strictly better for RPO but requires running
  and operating the receiver — which is exactly the decision we wanted to
  defer to the user.

## Consequences

- RPO is bounded by the cron schedule (typical: hours). Not suitable for
  teams that need sub-minute RPO.
- Restore is manual (`tsdb-ctl restore` + `kubectl cp` + pod restart). This
  is acceptable because restore is a rare event; documenting the runbook is
  more important than automating it.
- Users who need sub-minute RPO should configure `spec.remoteWrite` to a
  durable backend; that path is supported but not the default.

## Alternatives considered

1. **remote_write only, no snapshot feature.** Rejected: forces every user to
   operate a receiver.
2. **Velero / VolumeSnapshot-based PVC snapshots.** Rejected: couples backup
   integrity to CSI behavior, doesn't work uniformly across cloud providers,
   and fails the "off-cluster copy" requirement unless shipped to object
   storage anyway.
3. **Thanos sidecar.** Rejected as a default because it pulls the entire
   Thanos stack into the operational footprint. Left as a future opt-in
   (`spec.thanos.enabled`).
