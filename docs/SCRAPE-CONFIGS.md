# Adding custom scrape configs

中文版: [SCRAPE-CONFIGS.zh.md](SCRAPE-CONFIGS.zh.md)

The operator owns the `prometheus.yml` ConfigMap and rewrites it on every
reconcile. Hand-editing the ConfigMap will get clobbered. The supported
way to add custom scrape jobs is `spec.additionalScrapeConfigs`.

## How it works

The reconciler:

1. Stores the YAML you provide under the ConfigMap key
   `additional-scrape-configs.yml`.
2. Renders the main `prometheus.yml` with a
   [`scrape_config_files`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config_files)
   directive pointing at `/etc/prometheus/additional-scrape-configs.yml`.
3. Prometheus loads the file at startup and on
   `POST /-/reload`. Both the main and additional file are mounted from
   the same ConfigMap.

Requires Prometheus 2.43+ for the `scrape_config_files` directive (the
default image `prom/prometheus:v2.55.1` satisfies this).

## Two flavours

Since v0.11.0, `spec.additionalScrapeConfigs` is a struct with two
mutually-exclusive sub-fields. Use **inline** for short configs that
fit naturally in the CR; use **secretRef** when the config is large or
needs to be managed separately (RBAC, GitOps secrets, etc.).

### Inline

The operator stores it under ConfigMap key `additional-scrape-configs.yml`,
auto-wrapping under `scrape_configs:`. The user value is a bare YAML
list of scrape entries.

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: demo
spec:
  replicas: 1
  additionalScrapeConfigs:
    inline: |
      - job_name: my-app
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            action: keep
            regex: my-app
      - job_name: blackbox
        static_configs:
          - targets: [https://example.com]
        metrics_path: /probe
        params:
          module: [http_2xx]
```

### SecretRef

The Secret value must be a **complete Prometheus scrape config file**
(i.e. it includes the `scrape_configs:` top-level key — the operator
does not wrap secret content). The Secret is mounted at
`/etc/prometheus/extra-secret/<key>`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-scrape-secret
stringData:
  scrapes.yaml: |
    scrape_configs:
      - job_name: my-app
        static_configs:
          - targets: [my-app:8080]
---
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: demo
spec:
  replicas: 1
  additionalScrapeConfigs:
    secretRef:
      name: my-scrape-secret
      key: scrapes.yaml
```

## Validation

The admission webhook (when `features.webhook=true`) rejects:

- both `inline` and `secretRef` set
- neither set when `additionalScrapeConfigs` is present
- `inline` value that doesn't parse as a YAML list
- `secretRef.name` or `secretRef.key` empty

It does not validate every scrape-config field — Prometheus's reload
will still surface deeper errors via its log and the `/api/v1/status/config`
endpoint.

## Limitations

- **No PodMonitor / ServiceMonitor.** Those are prometheus-operator
  CRDs; this operator deliberately doesn't implement them. Pair both
  operators in the same cluster if you want both interfaces — see
  [`MIGRATION.md`](MIGRATION.md).
- **Reload is automatic since v0.10.1.** A `config-reloader` sidecar
  (`ghcr.io/jimmidyson/configmap-reload`) watches `/etc/prometheus`
  inside the pod and POSTs `/-/reload` when the mounted ConfigMap
  files change. This is the same pattern prometheus-operator uses and
  it sidesteps the kubelet ConfigMap projection lag (which broke the
  controller-driven approach attempted in v0.10.0).
