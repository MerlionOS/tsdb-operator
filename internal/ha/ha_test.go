package ha

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r), nil }

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := observabilityv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCheckClusterDeletesUnreadyPod(t *testing.T) {
	s := newScheme(t)
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-0", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
		Status: corev1.PodStatus{PodIP: "10.0.0.1"},
	}
	brokenPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-1", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
		Status: corev1.PodStatus{PodIP: "10.0.0.2"},
	}

	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(pc, healthyPod, brokenPod).
		WithStatusSubresource(&observabilityv1.PrometheusCluster{}).
		Build()

	rt := roundTripFunc(func(r *http.Request) *http.Response {
		code := http.StatusOK
		if strings.Contains(r.URL.Host, "10.0.0.2") {
			code = http.StatusInternalServerError
		}
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}
	})

	h := &HealthChecker{
		Client:   c,
		Recorder: record.NewFakeRecorder(10),
		HTTP:     &http.Client{Transport: rt},
	}
	h.checkCluster(context.Background(), pc)

	// Broken pod should have been deleted; healthy one should remain.
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "ns", Name: "demo-1"}, &corev1.Pod{}); err == nil {
		t.Fatal("expected broken pod to be deleted")
	}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "ns", Name: "demo-0"}, &corev1.Pod{}); err != nil {
		t.Fatalf("expected healthy pod to remain: %v", err)
	}

	// LastFailoverTime should be stamped.
	var updated observabilityv1.PrometheusCluster
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "ns", Name: "demo"}, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.LastFailoverTime == nil {
		t.Fatal("expected LastFailoverTime to be set")
	}
}

func TestCheckClusterSkipsPodsWithoutIP(t *testing.T) {
	s := newScheme(t)
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	pendingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-0", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(pc, pendingPod).
		WithStatusSubresource(&observabilityv1.PrometheusCluster{}).
		Build()

	called := false
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		called = true
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}
	})
	h := &HealthChecker{
		Client:   c,
		Recorder: record.NewFakeRecorder(1),
		HTTP:     &http.Client{Transport: rt},
	}
	h.checkCluster(context.Background(), pc)
	if called {
		t.Fatal("should not probe a pod with no PodIP")
	}
}
