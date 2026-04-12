# Migration guide

中文版: [MIGRATION.zh.md](MIGRATION.zh.md)

How to move between `tsdb-operator` and `prometheus-operator` in either
direction. The two operators own **different concerns** — scrape config
vs cluster lifecycle — so in most cases the right answer is to run both,
not migrate. This guide covers the cases where you genuinely want to
replace one with the other.

## Before you start

Read [ADR-0001](adr/0001-why-tsdb-operator-separate-from-prometheus-operator.md)
for the scope difference. If your need is *"I want declarative scrape
config"*, stay on prometheus-operator. If your need is *"I want HA,
scheduled S3 backups, audit, and a REST API around cluster lifecycle"*,
add tsdb-operator — you don't have to replace anything.

## Common prerequisite steps

1. Identify the source CRs. Snapshot them to YAML:
   ```bash
   kubectl get -A prometheuses.monitoring.coreos.com -o yaml > /tmp/from-po.yaml
   kubectl get -A servicemonitors.monitoring.coreos.com -o yaml > /tmp/from-po-sm.yaml
   ```
2. Back up the current TSDB(s) off-cluster before touching anything:
   ```bash
   kubectl exec -n <ns> <prometheus-pod> -- \
     curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot
   # then copy /prometheus/snapshots/<ts>/ off the pod
   ```
3. Quiesce scrapers that write into the cluster (`remote_write` sources,
   pushgateways) so no new samples land during the cutover.

## prometheus-operator → tsdb-operator

Do this when you specifically want the lifecycle / HA / backup / audit
features and are willing to rewrite scrape configuration.

### Step 1 — Install tsdb-operator alongside

```bash
helm install tsdb-operator ./charts/tsdb-operator \
  -n tsdb-operator --create-namespace
```

Both operators can run at the same time. The CRDs do not overlap
(`Prometheus` vs `PrometheusCluster`).

### Step 2 — Translate the `Prometheus` CR

Map fields — this table covers the common ones; adjust for your spec.

| prometheus-operator (`Prometheus`)    | tsdb-operator (`PrometheusCluster`)               |
|---------------------------------------|---------------------------------------------------|
| `spec.replicas`                       | `spec.replicas`                                   |
| `spec.image`                          | `spec.image`                                      |
| `spec.retention`                      | `spec.retention`                                  |
| `spec.resources`                      | `spec.resources`                                  |
| `spec.storage.volumeClaimTemplate`    | `spec.storage.size` + `storageClassName`          |
| `spec.remoteWrite[].url` + auth       | `spec.remoteWrite[].url` + `basicAuth` / `bearerToken` secrets |
| `spec.thanos`                         | `spec.thanos.enabled` + `objectStorageConfigSecretRef` |
| `spec.serviceMonitorSelector`         | *not supported* — scrape config is a user concern |
| `spec.podMonitorSelector`             | *not supported*                                   |
| `spec.ruleSelector`                   | *not supported* — use recording rules in `prometheus.yml` directly |

### Step 3 — Bring scrape config under your control

`tsdb-operator` owns the ConfigMap behind `prometheus.yml`. It installs a
default; patch your `ServiceMonitor`/`PodMonitor` rules into that
ConfigMap as plain Prometheus scrape config. The operator preserves
user edits to any key **other** than the default it seeds.

Alternative: keep prometheus-operator alive just for `ServiceMonitor`
CRDs and use its `promtool`-generated config as input. This is the
"run both" option mentioned in ADR-0001.

### Step 4 — Stop the old prometheus-operator managed Prometheus

```bash
kubectl -n <ns> delete prometheus <name>
```

This removes only the Prometheus StatefulSet. `ServiceMonitor`,
`PodMonitor`, and `PrometheusRule` CRs are left behind; delete or keep
per your choice.

### Step 5 — Restore from snapshot (optional)

If you backed up the TSDB in the prerequisite step:

```bash
kubectl cp /path/to/snapshot.tar <ns>/<new-cluster>-0:/prometheus/
kubectl exec -n <ns> <new-cluster>-0 -- tar -xf /prometheus/snapshot.tar -C /prometheus/
kubectl delete pod -n <ns> <new-cluster>-0
```

See [`RESTORE.md`](RESTORE.md) for the full runbook.

## tsdb-operator → prometheus-operator

Do this when you specifically need `ServiceMonitor`/`PodMonitor` as your
primary scrape-config surface, or kube-prometheus-stack as a whole.

### Step 1 — Install prometheus-operator

```bash
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace
```

### Step 2 — Translate the `PrometheusCluster` CR

| tsdb-operator (`PrometheusCluster`)  | prometheus-operator (`Prometheus`) |
|--------------------------------------|------------------------------------|
| `spec.replicas` / `image` / `retention` / `resources` | same                      |
| `spec.storage.size`                  | `spec.storage.volumeClaimTemplate.spec.resources.requests.storage` |
| `spec.remoteWrite`                   | `spec.remoteWrite`                 |
| `spec.thanos.enabled`                | `spec.thanos` block                |
| `spec.backup`                        | *no native equivalent* — replace with `velero`, `vmbackup`, or keep tsdb-operator running just for backups |
| `spec.backup` + audit log            | *no native equivalent*             |

### Step 3 — Author `ServiceMonitor` CRs

This is the whole reason you're migrating. Move each scrape block from
your `prometheus.yml` into a `ServiceMonitor` or `PodMonitor`.

### Step 4 — Stop tsdb-operator (or don't)

If you only migrated the Prometheus cluster itself but still want backups
and audit, leave tsdb-operator's REST API running against a secondary
`PrometheusCluster`. If not:

```bash
helm uninstall tsdb-operator -n tsdb-operator
kubectl delete ns tsdb-operator
kubectl delete crd prometheusclusters.observability.merlionos.org
```

### Step 5 — Restore data

Same as the forward direction: copy a snapshot into the new pod, extract
into `/prometheus`, restart.

## When things go wrong

- **Duplicate Prometheus replicas for the same workload.** Both operators
  are running and you didn't delete the old side. Confirm via `kubectl
  get pods -l app.kubernetes.io/name=prometheus -A`.
- **Grafana data source suddenly has gaps.** You forgot to restore the
  snapshot, or the target pod's PVC was re-created with a different
  StorageClass. Check `kubectl get pvc -n <ns>`.
- **`ServiceMonitor` scrapes that silently stopped working.** You moved
  to tsdb-operator but didn't migrate the scrape rules into
  `prometheus.yml`. Port them or run prometheus-operator in parallel.
