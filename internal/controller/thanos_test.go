package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

func TestThanosSidecarDisabled(t *testing.T) {
	r := &PrometheusClusterReconciler{}
	sts := r.buildStatefulSet(&observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
	})
	if got := len(sts.Spec.Template.Spec.Containers); got != 1 {
		t.Fatalf("want 1 container, got %d", got)
	}
	if sts.Spec.Template.Spec.Containers[0].Name != "prometheus" {
		t.Fatalf("unexpected container: %s", sts.Spec.Template.Spec.Containers[0].Name)
	}
}

func TestThanosEnabledDisablesPromCompaction(t *testing.T) {
	r := &PrometheusClusterReconciler{}
	sts := r.buildStatefulSet(&observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 1,
			Thanos:   observabilityv1.ThanosSpec{Enabled: true},
		},
	})
	args := sts.Spec.Template.Spec.Containers[0].Args
	var sawMin, sawMax bool
	for _, a := range args {
		if a == "--storage.tsdb.min-block-duration=2h" {
			sawMin = true
		}
		if a == "--storage.tsdb.max-block-duration=2h" {
			sawMax = true
		}
	}
	if !sawMin || !sawMax {
		t.Fatalf("thanos sidecar requires compaction disabled; args: %v", args)
	}
}

func TestThanosSidecarEnabledNoObjstore(t *testing.T) {
	r := &PrometheusClusterReconciler{}
	sts := r.buildStatefulSet(&observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 1,
			Thanos:   observabilityv1.ThanosSpec{Enabled: true},
		},
	})
	containers := sts.Spec.Template.Spec.Containers
	if len(containers) != 2 {
		t.Fatalf("want 2 containers, got %d", len(containers))
	}
	sidecar := containers[1]
	if sidecar.Name != "thanos-sidecar" {
		t.Errorf("second container = %q, want thanos-sidecar", sidecar.Name)
	}
	// Shares the data volume with Prometheus.
	var hasData bool
	for _, m := range sidecar.VolumeMounts {
		if m.Name == "data" && m.MountPath == "/prometheus" {
			hasData = true
		}
	}
	if !hasData {
		t.Error("thanos sidecar should mount the data volume at /prometheus")
	}
	// No objstore secret → no --objstore.config-file and no extra volume.
	for _, a := range sidecar.Args {
		if a == "--objstore.config-file=/etc/thanos/objstore/objstore.yml" {
			t.Error("unexpected --objstore.config-file when no secret ref is set")
		}
	}
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == "thanos-objstore" {
			t.Error("unexpected thanos-objstore volume when no secret ref is set")
		}
	}
}

func TestThanosSidecarEnabledWithObjstore(t *testing.T) {
	r := &PrometheusClusterReconciler{}
	sts := r.buildStatefulSet(&observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 1,
			Thanos: observabilityv1.ThanosSpec{
				Enabled:                      true,
				Image:                        "quay.io/thanos/thanos:v0.36.1",
				ObjectStorageConfigSecretRef: &corev1.LocalObjectReference{Name: "thanos-objstore-secret"},
			},
		},
	})
	sidecar := sts.Spec.Template.Spec.Containers[1]
	var sawFlag bool
	for _, a := range sidecar.Args {
		if a == "--objstore.config-file=/etc/thanos/objstore/objstore.yml" {
			sawFlag = true
		}
	}
	if !sawFlag {
		t.Error("--objstore.config-file missing from sidecar args")
	}
	var sawVol bool
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == "thanos-objstore" && v.Secret != nil && v.Secret.SecretName == "thanos-objstore-secret" {
			sawVol = true
		}
	}
	if !sawVol {
		t.Error("thanos-objstore volume not wired to the referenced Secret")
	}
}
