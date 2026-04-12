# TSDB Landscape

中文版: [TSDB-LANDSCAPE.zh.md](TSDB-LANDSCAPE.zh.md)

A horizontal overview of time-series databases you're likely to meet when
picking a backend for `tsdb-operator` or the wider observability stack.
Split into two groups: the **Prometheus-compatible** world (what this
operator lives in) and **general-purpose TSDBs** (useful to know, different
data models).

## 1. Prometheus-ecosystem (PromQL-compatible)

| | Prometheus | Thanos | Cortex | Grafana Mimir | VictoriaMetrics |
|---|---|---|---|---|---|
| **Origin** | Prometheus community (2012) | Improbable → CNCF | Weaveworks → CNCF (archived) | Grafana Labs (fork of Cortex) | VictoriaMetrics Inc. |
| **Scale model** | Single node (vertical) | Sidecar + object-storage fan-out | Microservices, horizontally sharded | Microservices, horizontally sharded | `vmstorage` sharding + replication |
| **Long-term storage** | Local TSDB only (retention-bounded) | Object storage (S3/GCS/Azure) | Object storage | Object storage | Local disk (cluster edition) or `vmbackup` to S3 |
| **Query language** | PromQL | PromQL (Thanos Query) | PromQL | PromQL | MetricsQL (PromQL-compatible superset) |
| **Multi-tenancy** | No | Limited | Yes (first-class) | Yes (first-class) | Yes (cluster edition) |
| **HA / dedup** | Run 2 replicas | Thanos Query dedup | Replicated ingesters | Replicated ingesters | `-replicationFactor` |
| **Typical ingest** | 100k–1M samples/s/node | Driven by underlying Prometheus | 10M+ samples/s (horizontal) | 10M+ samples/s (horizontal) | 10M+ samples/s; best density/RAM in the group |
| **License** | Apache 2.0 | Apache 2.0 | Apache 2.0 (archived) | AGPL-3.0 | Apache 2.0 (core); enterprise features BSL |
| **Managed offerings** | AWS AMP, GCP, Grafana Cloud | Red Hat OpenShift, community | (archived) | Grafana Cloud | VictoriaMetrics Cloud |

Notes:
- **Cortex** is effectively superseded by **Grafana Mimir** for new
  deployments; Mimir was forked with the express goal of cleaning up Cortex's
  operational rough edges.
- **Thanos vs Mimir/VM** is the big architectural split: Thanos keeps
  Prometheus as the ingestion engine and adds a read path; Mimir and VM run
  their own ingesters.

## 2. General-purpose TSDBs

| | InfluxDB 3 (IOx) | TimescaleDB | QuestDB | TDengine | ClickHouse | M3DB |
|---|---|---|---|---|---|---|
| **Storage engine** | Apache Parquet + Arrow (columnar) | PostgreSQL + hypertables | Columnar, memory-mapped | Custom columnar | Columnar (MergeTree) | Custom, M3TSZ compression |
| **Query language** | InfluxQL + SQL (+ Flux legacy) | SQL (Postgres dialect) | SQL | SQL | SQL | M3 Query / PromQL via M3Coordinator |
| **Primary use** | Observability + IoT | Observability + analytics within Postgres | Low-latency finance / IoT | IoT / industrial | General analytics (often TSDB in practice) | Uber-scale metrics |
| **Scale model** | Cloud-native object-storage | Vertical + multi-node cluster | Single node (horizontal WIP) | Cluster | Sharded cluster | Cluster with replicated shards |
| **Cardinality** | Very high (columnar) | High | Moderate | High | Very high | High |
| **Typical ingest** | Millions samples/s (cluster) | 100k–1M/s per node | Millions/s single node | Millions/s cluster | 10M+/s cluster | 10M+/s cluster |
| **License** | MIT (core v3) / commercial Cloud | Apache 2.0 + TSL for some features | Apache 2.0 | AGPL-3.0 + commercial | Apache 2.0 | Apache 2.0 |
| **Managed offerings** | InfluxDB Cloud | Timescale Cloud, Managed on AWS | QuestDB Cloud | TDengine Cloud | ClickHouse Cloud, Altinity | None (Uber internal) |

Notes:
- **InfluxDB 3 (IOx)** is a substantial rewrite in Rust using Parquet — very
  different from InfluxDB 1/2. If you read old comparisons, they probably
  don't apply.
- **ClickHouse** is not a "TSDB" strictly, but is heavily used as one
  (e.g. Uber M3 → ClickHouse migration, SigNoz, PostHog).
- **TimescaleDB** is great when you want **SQL + joins across time series
  and relational data** in the same database.

## Picking heuristics

- **"I just want Prometheus to work in Kubernetes"** → Prometheus +
  `tsdb-operator` + Alertmanager. Add Thanos later if you outgrow it.
- **"I need global query across 20+ Prometheus"** → Thanos or Mimir.
- **"I want the highest ingest per dollar on the same PromQL-ish API"** →
  VictoriaMetrics cluster.
- **"I want SQL and joins against my business tables"** → TimescaleDB.
- **"I'm building a product with high-cardinality tracing/events alongside
  metrics"** → ClickHouse or InfluxDB 3.
- **"IoT device fleet, edge-constrained"** → TDengine or QuestDB.

## What this operator is orthogonal to

`tsdb-operator` manages the **Prometheus process lifecycle**. The choice of
long-term backend (Thanos vs Mimir vs VM vs just S3 snapshots) is a separate
decision — pick the one whose operational footprint matches your team's size
and the query patterns your users actually need.
