package controller

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r), nil }

func TestTriggerReloadHitsEveryReadyPod(t *testing.T) {
	pc := &observabilityv1.PrometheusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	pods := []*corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-0", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
		Status: corev1.PodStatus{PodIP: "10.0.0.1"},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-1", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
		Status: corev1.PodStatus{PodIP: "10.0.0.2"},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-2", Namespace: "ns",
			Labels: map[string]string{"app.kubernetes.io/instance": "demo"},
		},
		// No PodIP — should be skipped.
	}}

	objs := make([]runtimeObject, 0, len(pods))
	for _, p := range pods {
		objs = append(objs, p)
	}
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(toClientObjects(objs)...).Build()

	var hits atomic.Int32
	rt := roundTripFunc(func(r *http.Request) *http.Response {
		if !strings.HasSuffix(r.URL.Path, "/-/reload") || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		hits.Add(1)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}
	})
	r := &PrometheusClusterReconciler{
		Client: c,
		Scheme: c.Scheme(),
		HTTP:   &http.Client{Transport: rt},
	}
	r.triggerReload(context.Background(), pc)
	if got := hits.Load(); got != 2 {
		t.Fatalf("want 2 reload calls (skip the pod with no IP), got %d", got)
	}
}
