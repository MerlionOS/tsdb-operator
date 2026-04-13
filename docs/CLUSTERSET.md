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

## What this release does NOT do

- **Auto-patch** `backupTemplate` into matched `PrometheusCluster`
  resources. The field is recorded in the spec but member CRs still need
  their own `spec.backup` filled in. Auto-overlay is planned for a
  follow-up release; this lets users see the policy intent without the
  Set silently mutating their CRs.
- Cross-cluster (multi-Kubernetes) federation. Single Kubernetes cluster
  only.
