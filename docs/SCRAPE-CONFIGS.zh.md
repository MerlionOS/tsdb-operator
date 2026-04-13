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
`prom/prometheus:v2.55.1` 满足）。

## 两种形式

v0.11.0 起 `spec.additionalScrapeConfigs` 是一个结构体，两个互斥的
子字段：短配置用 **inline** 直接写在 CR 里；长配置或要分开管理（RBAC、
GitOps secret 等）用 **secretRef**。

### Inline

operator 存到 ConfigMap 的 `additional-scrape-configs.yml` key，
自动包一层 `scrape_configs:`。用户写裸 list。

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

Secret 里的内容必须是**完整的 Prometheus 抓取配置文件**（包含顶层
`scrape_configs:` —— operator 不会自动包 secret 内容）。Secret
被挂到 `/etc/prometheus/extra-secret/<key>`。

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

## 校验

启用 `features.webhook=true` 时 admission webhook 拒绝以下情况：

- `inline` 和 `secretRef` 同时设置
- `additionalScrapeConfigs` 存在但两者都没设
- `inline` 不是 YAML list
- `secretRef.name` 或 `secretRef.key` 为空

它**不会**校验每个 scrape-config 字段的细节 —— 那种深度的错误还是会
通过 Prometheus reload 日志和 `/api/v1/status/config` 暴露出来。

## 局限

- **不支持 PodMonitor / ServiceMonitor。** 那是 prometheus-operator
  的 CRD，本 operator 故意不实现。同时跑两个 operator 即可同时拥有
  两套接口 —— 见 [`MIGRATION.zh.md`](MIGRATION.zh.md)。
- **v0.10.1 起 reload 是自动的。** Pod 里跑一个
  `config-reloader` sidecar（`ghcr.io/jimmidyson/configmap-reload`）
  watch `/etc/prometheus`，挂载的 ConfigMap 文件变化时 POST
  `/-/reload`。和 prometheus-operator 同款方案，绕开 kubelet 的
  ConfigMap projection 延迟（这个延迟搞挂了 v0.10.0 的 controller
  驱动方案）。
