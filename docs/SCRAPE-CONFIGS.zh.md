# 加自定义抓取配置

English: [SCRAPE-CONFIGS.md](SCRAPE-CONFIGS.md)

operator 拥有 `prometheus.yml` 这个 ConfigMap，每次 reconcile 都会
覆写。直接手改 ConfigMap 会被覆盖。加自定义抓取任务的官方方式是
`spec.additionalScrapeConfigs`。

## 工作方式

reconciler：

1. 把你给的 YAML 写到 ConfigMap 的 `additional-scrape-configs.yml` 这个 key。
2. 在主 `prometheus.yml` 里渲染一条
   [`scrape_config_files`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config_files)
   指向 `/etc/prometheus/additional-scrape-configs.yml`。
3. Prometheus 启动时和收到 `POST /-/reload` 时加载该文件。主文件和
   附加文件都是同一个 ConfigMap 挂出来的。

需要 Prometheus 2.43+ 才支持 `scrape_config_files` 指令（默认镜像
`prom/prometheus:v2.53.0` 满足）。

## 示例

```yaml
apiVersion: observability.merlionos.org/v1
kind: PrometheusCluster
metadata:
  name: demo
spec:
  replicas: 1
  additionalScrapeConfigs: |
    - job_name: my-app
      kubernetes_sd_configs:
        - role: pod
      relabel_configs:
        - source_labels: [__meta_kubernetes_pod_label_app]
          action: keep
          regex: my-app
    - job_name: blackbox
      static_configs:
        - targets:
            - https://example.com
      metrics_path: /probe
      params:
        module: [http_2xx]
```

值是一个**顶层 YAML 列表**，和主配置 `scrape_configs:` 下面的形状
完全一样。

## 校验

启用 `features.webhook=true` 时，admission webhook 在 `kubectl apply`
阶段就会解析这个字段，不是 YAML list 直接拒。它**不会**校验每个
scrape-config 字段的细节 —— 那种深度的错误还是会通过 Prometheus
reload 日志和 `/api/v1/status/config` 暴露出来。

## 局限

- **v0.9.0 只支持 inline。** `secretRef` 形式（挂一个装满 scrape file
  的 Secret）是自然的后续，本版没做。
- **不支持 PodMonitor / ServiceMonitor。** 那是 prometheus-operator
  的 CRD，本 operator 故意不实现。同时跑两个 operator 即可同时拥有
  两套接口 —— 见 [`MIGRATION.zh.md`](MIGRATION.zh.md)。
- **ConfigMap 变更不会自动 reload。** Prometheus 默认开了
  `--web.enable-lifecycle`，需要手动
  `kubectl exec <pod> -- curl -XPOST http://localhost:9090/-/reload`
  或者等下次 Pod 重启。operator 自动 reload 留在 Later 列表。
