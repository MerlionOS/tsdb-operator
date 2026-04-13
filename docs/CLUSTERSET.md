# PrometheusClusterSet

中文版: [CLUSTERSET.zh.md](CLUSTERSET.zh.md)

A `PrometheusClusterSet` is a **cluster-scoped** resource that groups
`PrometheusCluster`s by label across namespaces. It does not create
Prometheus instances of its own — it observes membership and (optionally)
carries default policy that members can inherit.

## Why

In a real platform, "the observability team" owns a fleet of Prometheus
clusters spread across product-team namespaces. Without a higher-level
object, listing them, reporting their phases, and giving them a single
default backup policy means scripting on the side. `PrometheusClusterSet`
is that higher-level object.

## Example

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusClusterSet
metadata:
  name: tier1
spec:
  clusterSelector:
    matchLabels:
      tier: t1
  namespaceSelector:
    matchLabels:
      env: prod
  backupTemplate:
    enabled: true
    bucket: tsdb-tier1-backups
    schedule: "0 */4 * * *"
```

This selects every `PrometheusCluster` labelled `tier=t1` in any
namespace labelled `env=prod`.

## Status

The reconciler updates the Set's status with the membership and a per-
phase histogram:

```yaml
status:
  memberCount: 3
  phaseCount:
    Active: 2
    Provisioning: 1
  members:
    - namespace: team-a
      name: prom-a
      phase: Active
    - namespace: team-b
      name: prom-b
      phase: Active
    - namespace: team-c
      name: prom-c
      phase: Provisioning
```

The Set re-reconciles whenever any matching `PrometheusCluster` changes
(via a `Watches` predicate that enqueues every Set on every
`PrometheusCluster` event — fine because Set cardinality is small).

## REST API

| Method | Path                          | Description              |
|--------|-------------------------------|--------------------------|
| GET    | `/api/clustersets`            | List all Sets            |
| GET    | `/api/clustersets/:name`      | Get one Set with status  |

## `backupTemplate` overlay (v0.8.0+)

When a Set carries a `spec.backupTemplate`, the Set reconciler copies it
onto matched members under the following rules, in order:

1. **Template must be set.** No-op if `spec.backupTemplate` is nil.
2. **Opt-out wins.** A member with annotation
   `observability.merlionos.org/clusterset-opt-out: "true"` is never
   touched.
3. **Explicit member wins.** A member whose `spec.backup.enabled` is
   already `true` keeps its own config.
4. **Otherwise**: the member's `spec.backup` is replaced wholesale by
   the template (with `enabled: true`), and an annotation
   `observability.merlionos.org/clusterset: <set-name>` is stamped for
   traceability.

The `PrometheusCluster` reconciler picks up the mutation through its own
watch, so the backup scheduler registers the cron entry on the next
reconcile.

### Notes

- This is **all-or-nothing per member**. We don't do field-level merge
  (Go zero values are indistinguishable from "unset" without pointer
  plumbing on every backup field). Members either fully inherit or
  fully own.
- Deleting the Set **does not** remove the overlay from its former
  members. That's deliberate — silently disabling backups on delete is
  a bigger footgun than leaving them running.
- Setting `enabled: true` on a member after an overlay transfers
  ownership back to the user: the next Set reconcile leaves it alone.

## What this release does NOT do

- Cross-cluster (multi-Kubernetes) federation. Single Kubernetes cluster
  only.
