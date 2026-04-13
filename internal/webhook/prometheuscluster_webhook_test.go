package webhook

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

func TestValidateRejectsZeroReplicas(t *testing.T) {
	v := &PrometheusClusterValidator{}
	err := v.Validate(&observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec:       observabilityv1.PrometheusClusterSpec{Replicas: 0},
	})
	if err == nil || !strings.Contains(err.Error(), "replicas") {
		t.Fatalf("want replicas error, got %v", err)
	}
}

func TestValidateRejectsBadCron(t *testing.T) {
	v := &PrometheusClusterValidator{}
	err := v.Validate(&observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 1,
			Backup: observabilityv1.S3BackupSpec{
				Enabled: true, Bucket: "b", Schedule: "not a cron",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "schedule") {
		t.Fatalf("want schedule error, got %v", err)
	}
}

func TestValidateRejectsMissingBucketWhenBackupEnabled(t *testing.T) {
	v := &PrometheusClusterValidator{}
	err := v.Validate(&observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 1,
			Backup:   observabilityv1.S3BackupSpec{Enabled: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("want bucket error, got %v", err)
	}
}

func TestValidateRejectsRemoteWriteWithoutURL(t *testing.T) {
	v := &PrometheusClusterValidator{}
	err := v.Validate(&observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas:    1,
			RemoteWrite: []observabilityv1.RemoteWriteSpec{{URL: ""}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Fatalf("want url error, got %v", err)
	}
}

func TestValidateAcceptsValid(t *testing.T) {
	v := &PrometheusClusterValidator{}
	err := v.Validate(&observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: observabilityv1.PrometheusClusterSpec{
			Replicas: 2,
			Backup: observabilityv1.S3BackupSpec{
				Enabled: true, Bucket: "b", Schedule: "0 */6 * * *",
			},
		},
	})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}
