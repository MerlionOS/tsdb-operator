# ADR 0001: Why tsdb-operator is separate from prometheus-operator

- Status: Accepted
- Date: 2026-04-12

## Context

`prometheus-operator` is the de-facto Kubernetes operator for running Prometheus.
It exposes `Prometheus`, `ServiceMonitor`, `PodMonitor`, `PrometheusRule`, etc.,
and is bundled into kube-prometheus-stack. It was designed for one concern:
**declarative scrape configuration**.

Many platform teams independently re-implement a different concern on top of it:
**cluster lifecycle and Day-2 operations** — provisioning, HA failover, scheduled
off-cluster backups, and an audit trail of who changed what. This ADR records
why we solve that separately rather than as a fork or a PR against
prometheus-operator.

## Decision

Build `tsdb-operator` as an independent operator that is **complementary** to
prometheus-operator, not a replacement:

- `prometheus-operator` owns *what gets scraped* — scrape config, recording
  rules, alerting rules.
- `tsdb-operator` owns *how the Prometheus cluster itself is provisioned,
  backed up, and audited*.

Both can run in the same cluster. A `PrometheusCluster` from tsdb-operator and
a `ServiceMonitor` from prometheus-operator don't conflict — they address
different CRDs and different phases of the life of a Prometheus instance.

## Consequences

- Users who only need scrape config continue with prometheus-operator; they
  pay no cost.
- Users who need HA + backup + audit + a REST control plane install
  tsdb-operator in addition — they do not have to fork upstream.
- We inherit upstream Prometheus unchanged. No custom binary, no patches.
- Two operators means two sets of CRDs and RBAC. Worth the clarity.

## Alternatives considered

1. **Upstream the features into prometheus-operator.** Rejected: scope creep,
   and the maintainers have consistently kept it focused on scrape config.
2. **Fork prometheus-operator.** Rejected: forking a widely-adopted operator
   fragments the ecosystem and creates ongoing merge pain.
3. **Bash scripts + Helm hooks.** Works for a single cluster; does not model
   reconciliation, does not survive a control-plane outage, does not produce
   an audit trail.
