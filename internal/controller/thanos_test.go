package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// containerByName returns the container with the given name, or nil.
func containerByName(cs []corev1.Container, name string) *corev1.Container {
	for i := range cs {
		if cs[i].Name == name {
			return &cs[i]
		}
	}
	return nil
}

func TestThanosSidecarDisabled(t *testing.T) {
	r := &PrometheusClusterReconciler{}
	sts := r.buildStatefulSet(&observabilityv1.PrometheusCluster{
		Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
	})
	cs := sts.Spec.Template.Spec.Containers
	if got := len(cs); got != 2 {
		t.Fatalf("want 2 containers (prometheus + reloader), got %d", got)
	}
	if containerByName(cs, "prometheus") == nil {
		t.Error("missing prometheus container")
	}
	if containerByName(cs, "config-reloader") == nil {
		t.Error("missing config-reloader sidecar")
	}
	if containerByName(cs, "thanos-sidecar") != nil {
		t.Error("unexpected thanos-sidecar when disabled")
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
	prom := containerByName(sts.Spec.Template.Spec.Containers, "prometheus")
	if prom == nil {
		t.Fatal("prometheus container missing")
	}
	var sawMin, sawMax bool
	for _, a := range prom.Args {
		if a == "--storage.tsdb.min-block-duration=2h" {
			sawMin = true
		}
		if a == "--storage.tsdb.max-block-duration=2h" {
			sawMax = true
		}
	}
	if !sawMin || !sawMax {
		t.Fatalf("thanos sidecar requires compaction disabled; args: %v", prom.Args)
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
	sidecar := containerByName(sts.Spec.Template.Spec.Containers, "thanos-sidecar")
	if sidecar == nil {
		t.Fatal("thanos-sidecar container missing")
	}
	var hasData bool
	for _, m := range sidecar.VolumeMounts {
		if m.Name == "data" && m.MountPath == "/prometheus" {
			hasData = true
		}
	}
	if !hasData {
		t.Error("thanos sidecar should mount the data volume at /prometheus")
	}
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
				Image:                        "quay.io/thanos/thanos:v0.37.2",
				ObjectStorageConfigSecretRef: &corev1.LocalObjectReference{Name: "thanos-objstore-secret"},
			},
		},
	})
	sidecar := containerByName(sts.Spec.Template.Spec.Containers, "thanos-sidecar")
	if sidecar == nil {
		t.Fatal("thanos-sidecar container missing")
	}
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
