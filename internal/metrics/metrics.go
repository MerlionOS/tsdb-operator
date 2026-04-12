// Package metrics declares the Prometheus metrics emitted by tsdb-operator
// and registers them with controller-runtime's metrics registry so they are
// exposed on the manager's /metrics endpoint.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ClusterPhase is 1 for the phase a cluster is currently in, 0 otherwise.
	// The cardinality is bounded by the number of known phases.
	ClusterPhase = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tsdb_operator_cluster_phase",
		Help: "Current lifecycle phase of a PrometheusCluster (1 = active phase).",
	}, []string{"namespace", "cluster", "phase"})

	// BackupTotal counts backup attempts by result (success or error).
	BackupTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tsdb_operator_backup_total",
		Help: "Total number of backup attempts by result.",
	}, []string{"namespace", "cluster", "result"})

	// FailoverTotal counts failover events per cluster.
	FailoverTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tsdb_operator_failover_total",
		Help: "Total number of replica failovers triggered.",
	}, []string{"namespace", "cluster"})
)

// Phases mirrors the enum in api/v1 so SetPhase can zero the others.
var Phases = []string{"Provisioning", "Active", "Scaling", "Failed"}

// SetPhase records that cluster is in phase by setting the matching gauge to
// 1 and all other known phases to 0. Keeps dashboards honest without needing
// per-phase Delete logic on transitions.
func SetPhase(namespace, cluster, phase string) {
	for _, p := range Phases {
		v := 0.0
		if p == phase {
			v = 1.0
		}
		ClusterPhase.WithLabelValues(namespace, cluster, p).Set(v)
	}
}

// DeleteCluster removes all series for a cluster (e.g. on delete).
func DeleteCluster(namespace, cluster string) {
	ClusterPhase.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "cluster": cluster})
	BackupTotal.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "cluster": cluster})
	FailoverTotal.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "cluster": cluster})
}

func init() {
	metrics.Registry.MustRegister(ClusterPhase, BackupTotal, FailoverTotal)
}
