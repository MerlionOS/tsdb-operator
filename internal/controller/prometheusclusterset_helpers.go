package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// corev1NamespaceList is a thin alias so the reconciler stays decoupled
// from the corev1 import in its main file.
type corev1NamespaceList = corev1.NamespaceList

// enqueueAllSets returns a handler that, on any PrometheusCluster change,
// enqueues every PrometheusClusterSet so the Set reconciler refreshes its
// status. Cardinality is the number of sets, not the number of clusters,
// so this is fine.
func enqueueAllSets(c client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ client.Object) []reconcile.Request {
		var sets observabilityv1.PrometheusClusterSetList
		if err := c.List(ctx, &sets); err != nil {
			return nil
		}
		out := make([]reconcile.Request, 0, len(sets.Items))
		for _, s := range sets.Items {
			out = append(out, reconcile.Request{
				NamespacedName: client.ObjectKey{Name: s.Name},
			})
		}
		return out
	})
}
