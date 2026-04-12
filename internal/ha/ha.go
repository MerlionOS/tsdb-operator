// Package ha implements periodic health checking and failover for
// PrometheusCluster replicas managed by tsdb-operator.
package ha

import (
	"context"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// HealthChecker polls replica /-/ready endpoints and records failovers.
type HealthChecker struct {
	Client   client.Client
	Recorder record.EventRecorder
	Interval time.Duration
	HTTP     *http.Client
}

// New builds a HealthChecker with sensible defaults.
func New(c client.Client, rec record.EventRecorder) *HealthChecker {
	return &HealthChecker{
		Client:   c,
		Recorder: rec,
		Interval: 30 * time.Second,
		HTTP:     &http.Client{Timeout: 5 * time.Second},
	}
}

// Start runs the health-check loop until ctx is cancelled.
func (h *HealthChecker) Start(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("ha")
	ticker := time.NewTicker(h.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := h.checkAll(ctx); err != nil {
				log.Error(err, "health check pass failed")
			}
		}
	}
}

func (h *HealthChecker) checkAll(ctx context.Context) error {
	var list observabilityv1.PrometheusClusterList
	if err := h.Client.List(ctx, &list); err != nil {
		return fmt.Errorf("list clusters: %w", err)
	}
	for i := range list.Items {
		h.checkCluster(ctx, &list.Items[i])
	}
	return nil
}

func (h *HealthChecker) checkCluster(ctx context.Context, pc *observabilityv1.PrometheusCluster) {
	log := logf.FromContext(ctx).WithValues("cluster", pc.Name)
	var pods corev1.PodList
	if err := h.Client.List(ctx, &pods, client.InNamespace(pc.Namespace), client.MatchingLabels{
		"app.kubernetes.io/instance": pc.Name,
	}); err != nil {
		log.Error(err, "list pods")
		return
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.PodIP == "" {
			continue
		}
		url := fmt.Sprintf("http://%s:9090/-/ready", pod.Status.PodIP)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := h.HTTP.Do(req)
		if err != nil || resp.StatusCode >= 500 {
			h.triggerFailover(ctx, pc, pod, err)
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}
		_ = resp.Body.Close()
	}
}

func (h *HealthChecker) triggerFailover(ctx context.Context, pc *observabilityv1.PrometheusCluster, pod *corev1.Pod, cause error) {
	log := logf.FromContext(ctx).WithValues("cluster", pc.Name, "pod", pod.Name)
	reason := "Unreachable"
	if cause != nil {
		reason = cause.Error()
	}
	h.Recorder.Eventf(pc, corev1.EventTypeWarning, "FailoverTriggered",
		"replica %s failed health check: %s — deleting pod to trigger rescheduling", pod.Name, reason)
	if err := h.Client.Delete(ctx, pod); err != nil {
		log.Error(err, "delete failed pod")
		return
	}
	now := metav1.Now()
	pc.Status.LastFailoverTime = &now
	if err := h.Client.Status().Update(ctx, pc); err != nil {
		log.Error(err, "update LastFailoverTime")
	}
}
