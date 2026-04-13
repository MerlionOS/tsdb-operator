/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"
	"fmt"
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// PrometheusClusterSetReconciler watches PrometheusClusterSet resources and
// keeps their status in sync with the set of matching PrometheusCluster
// resources across all (or selected) namespaces.
type PrometheusClusterSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclustersets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclustersets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclustersets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (r *PrometheusClusterSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var set observabilityv1.PrometheusClusterSet
	if err := r.Get(ctx, req.NamespacedName, &set); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get clusterset: %w", err)
	}

	clusterSel, err := selectorFor(set.Spec.ClusterSelector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("invalid clusterSelector: %w", err)
	}
	nsSel, err := selectorFor(set.Spec.NamespaceSelector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("invalid namespaceSelector: %w", err)
	}

	allowedNS, err := r.allowedNamespaces(ctx, nsSel)
	if err != nil {
		return ctrl.Result{}, err
	}

	var pcList observabilityv1.PrometheusClusterList
	if err := r.List(ctx, &pcList); err != nil {
		return ctrl.Result{}, fmt.Errorf("list prometheusclusters: %w", err)
	}

	members := []observabilityv1.SetMember{}
	phaseCount := map[string]int32{}
	for i := range pcList.Items {
		pc := &pcList.Items[i]
		if allowedNS != nil {
			if _, ok := allowedNS[pc.Namespace]; !ok {
				continue
			}
		}
		if !clusterSel.Matches(labels.Set(pc.Labels)) {
			continue
		}
		members = append(members, observabilityv1.SetMember{
			Namespace: pc.Namespace,
			Name:      pc.Name,
			Phase:     pc.Status.Phase,
		})
		if pc.Status.Phase != "" {
			phaseCount[string(pc.Status.Phase)]++
		}
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].Namespace != members[j].Namespace {
			return members[i].Namespace < members[j].Namespace
		}
		return members[i].Name < members[j].Name
	})

	set.Status.MemberCount = int32(len(members))
	set.Status.Members = members
	set.Status.PhaseCount = phaseCount
	if err := r.Status().Update(ctx, &set); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}
	log.V(1).Info("reconciled clusterset", "members", len(members))
	return ctrl.Result{}, nil
}

// allowedNamespaces returns nil when the selector is everything, otherwise a
// set of namespace names that match.
func (r *PrometheusClusterSetReconciler) allowedNamespaces(ctx context.Context, sel labels.Selector) (map[string]struct{}, error) {
	if sel.Empty() {
		return nil, nil
	}
	var list corev1NamespaceList
	if err := r.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	out := map[string]struct{}{}
	for _, ns := range list.Items {
		if sel.Matches(labels.Set(ns.Labels)) {
			out[ns.Name] = struct{}{}
		}
	}
	return out, nil
}

func selectorFor(s *metav1.LabelSelector) (labels.Selector, error) {
	if s == nil {
		return labels.Everything(), nil
	}
	return metav1.LabelSelectorAsSelector(s)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusClusterSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.PrometheusClusterSet{}).
		Watches(
			&observabilityv1.PrometheusCluster{},
			enqueueAllSets(r.Client),
		).
		Named("prometheusclusterset").
		Complete(r)
}
