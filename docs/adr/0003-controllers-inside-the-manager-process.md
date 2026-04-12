# ADR 0003: HA checker and backup scheduler run inside the manager process

- Status: Accepted
- Date: 2026-04-12

## Context

`tsdb-operator` has three long-running control loops:

1. The `PrometheusCluster` reconciler (triggered by watch events).
2. The HA health checker (fixed-interval probe across all replicas).
3. The backup scheduler (cron-driven snapshot + S3 upload).

They could be deployed as three separate Deployments, or as threads inside
the controller-runtime manager process.

## Decision

All three run as `manager.Runnable`s inside the same binary and the same
pod. Registration happens in `cmd/main.go` behind feature flags
(`--enable-ha`, `--enable-backup`).

## Consequences

- Single RBAC surface, single set of metrics, single process to operate.
- Leader election (via controller-runtime) naturally extends to the other
  loops — the active leader is also the one probing and backing up, so we
  don't get duplicate failovers or duplicate uploads across replicas.
- If the manager pod crashes, everything restarts together. For a small
  operator this is desirable (atomic failure modes).
- Scaling out one concern (e.g. many clusters to probe) is harder because
  every concern scales together. Acceptable at this stage; if HA probing
  becomes a bottleneck, we will split it out and revisit this ADR.

## Alternatives considered

1. **Separate Deployments per concern.** Rejected as premature. More moving
   parts, more leader-election coordination, no current scaling pressure.
2. **External cron (CronJob objects) for backups.** Rejected: loses the
   shared audit/event path, and the operator no longer owns end-to-end
   backup success/failure semantics.
