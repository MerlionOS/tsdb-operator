# v1.0 preparation

中文版: [V1-PREP.zh.md](V1-PREP.zh.md)

This document captures the work needed to ship `v1.0`. v1 is not a
feature release — it's the moment we commit to API stability under
[semver](https://semver.org/) for `observability.merlionos.org/v1`.

> If you are reading this and are about to add a new field to a CRD,
> ask: "would I bet the next 12 months of breakage on this field
> name and shape?" If not, defer to v0.x.

## Why now

By v0.10.x the operator covers every Milestone-4 commitment plus the
high-value Later items. There is no obvious feature still pulling. The
pattern of finding a real bug on every kind verification (10 of 11
shipped versions did) has stabilised. Time to stop adding surface and
start guaranteeing it.

## API stability review

For each field on every CRD, we need a v1 verdict:

| Field                                    | v1 status | Notes |
|------------------------------------------|-----------|-------|
| `PrometheusCluster.spec.replicas`         | **stable** | int32, default 1, min 1 enforced by webhook + CRD schema |
| `PrometheusCluster.spec.image`            | **stable** | string, default to a pinned Prometheus minor |
| `PrometheusCluster.spec.retention`        | **stable** | string, validated as Prometheus duration |
| `PrometheusCluster.spec.storage.size`     | **stable** | resource.Quantity |
| `PrometheusCluster.spec.storage.storageClassName` | **stable** | `*string` |
| `PrometheusCluster.spec.resources`        | **stable** | `corev1.ResourceRequirements`, structural |
| `PrometheusCluster.spec.backup.*`         | **stable** | flat S3 fields; the only awkward shape is `endpoint` (MinIO escape hatch). Keep. |
| `PrometheusCluster.spec.remoteWrite[]`    | **stable** | `URL` required; auth via Secret refs only — intentional, no inline secrets |
| `PrometheusCluster.spec.thanos.*`         | **stable** | enable + image + objstore secret. No retention knob — Prometheus retention rules. |
| `PrometheusCluster.spec.additionalScrapeConfigs` | **review** | currently a string. See "Breaking-change inventory" below. |
| `PrometheusCluster.status.*`              | **stable** | phase enum is closed; LastBackupTime + LastFailoverTime are `*metav1.Time` |
| `PrometheusClusterSet.spec.clusterSelector` | **stable** | `*metav1.LabelSelector` |
| `PrometheusClusterSet.spec.namespaceSelector` | **stable** | `*metav1.LabelSelector` |
| `PrometheusClusterSet.spec.backupTemplate` | **stable** | `*S3BackupSpec`; overlay rules pinned in v0.8.0 docs |

## Breaking-change inventory

Things to consider changing **before** the v1 freeze:

1. **`additionalScrapeConfigs string` → `additionalScrapeConfigs.inline string` + `additionalScrapeConfigs.secretRef *LocalObjectReference`?**
   v0.9.x intentionally shipped only the inline form. If we promote the
   string to v1 as-is, adding a Secret variant later requires a new
   field name, and we end up with `additionalScrapeConfigs` (string)
   plus `additionalScrapeConfigsSecretRef` (ref) — awkward forever.
   Cleaner to ship v1 with a struct that has `inline` and `secretRef`
   subfields, mutually exclusive in the webhook.
   **Decision: yes, refactor before v1.**

2. **`status.lastBackupTime` is per-cluster, not per-replica.**
   When we run multiple replicas with shared backup later, we'd want a
   per-replica record. Today there's no consumer of "per-replica
   backup time" — leave as-is, can extend with `status.replicas[].lastBackup`
   later without breaking the existing field.
   **Decision: no change.**

3. **`PrometheusClusterSpec.Backup.Endpoint`** is a MinIO escape hatch
   that doesn't really belong on the user-facing CRD. But there's no
   cleaner place — operator-level `--s3-endpoint` exists but per-cluster
   override lives here.
   **Decision: keep, document as "for testing / on-prem object stores".**

4. **`spec.thanos.image` defaults to a pinned Thanos version (`v0.36.1`).**
   Make sure the default is fresh enough that v1 ships with a recent
   stable Thanos. Bump to current Thanos LTS at v1 cut.

5. **`spec.image` default** — same deal for Prometheus. Pin to the
   most recent 2.x at v1.

## Conversion webhook decision

Two paths:

- **Path A — promote in place.** v0.10.x schema becomes v1. Users on
  v0.10.x just upgrade the operator; CRs work unchanged. Requires
  resolving the `additionalScrapeConfigs` shape question above
  *before* v1 (no cleanup window after).
- **Path B — `v1` alongside `v1alpha1`.** Ship a conversion webhook,
  let v1 break the schema where it makes sense. More work; allows
  multiple shape changes; canonical for "real" v1 of an operator.

**Recommended: Path A**, contingent on the `additionalScrapeConfigs`
refactor (item 1) being the *only* breaking change. The rest of the
schema is genuinely fine. Conversion-webhook complexity isn't worth
paying for one field rename.

If a second breaking change emerges during the review, re-evaluate
toward Path B.

## Deprecation policy (post-v1)

Adopt the [Kubernetes API deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/)
in spirit:

- Fields go through `// Deprecated:` godoc + `+kubebuilder:deprecatedversion:warning`
  for one operator minor version before removal.
- Renames ship the new field for a release alongside the old; the old
  field's value is mirrored if both are set, with the new winning.
- Schema removals require a major bump (v1 → v2).

## What v1 does not promise

- Internal package layout (`internal/...`) is **not** stable. Importing
  these packages from outside the operator binary is unsupported.
- The REST API JSON shape is **not** part of the CRD semver guarantee
  — it tracks the CRD shape but may add fields freely.
- Helm chart values **track** but may add new opt-in keys without a
  major bump (additions are non-breaking).
- The audit log table schema is owned by the operator; no external
  reads.

## Checklist before tagging v1.0.0

- [x] `additionalScrapeConfigs` refactored to struct — **v0.11.0**
- [x] `// +kubebuilder:validation:*` markers reviewed: MinLength on
  `remoteWrite[].url` and `backup.schedule`, Pattern on `retention`
  (Prometheus duration), Enum on `status.phase`
- [x] Storage version: `+kubebuilder:storageversion` on both
  `PrometheusCluster` and `PrometheusClusterSet`
- [x] Print columns: `kubectl get prometheuscluster` now shows
  Phase / Ready / Age; `kubectl get prometheusclusterset` shows
  Members / Age
- [x] Default `image` bumped to `prom/prometheus:v2.55.1` and
  `spec.thanos.image` to `quay.io/thanos/thanos:v0.37.2`
- [x] CHANGELOG entry under `## [1.0.0]` with migration note (only CR
  edit needed is from v0.10.x or earlier, and it's `additionalScrapeConfigs`
  from v0.11.0)
