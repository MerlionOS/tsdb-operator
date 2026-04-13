/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

var _ = Describe("PrometheusClusterSet Controller", func() {
	const (
		setName = "test-set"
		nsA     = "default"
	)
	ctx := context.Background()

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
		})
		var pcs observabilityv1.PrometheusClusterList
		_ = k8sClient.List(ctx, &pcs)
		for i := range pcs.Items {
			_ = k8sClient.Delete(ctx, &pcs.Items[i])
		}
	})

	It("matches PrometheusClusters by label and reports counts", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "alpha", Namespace: nsA,
				Labels: map[string]string{"team": "obs"},
			},
			Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "beta", Namespace: nsA,
				Labels: map[string]string{"team": "platform"},
			},
			Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())

		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
			Spec: observabilityv1.PrometheusClusterSetSpec{
				ClusterSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"team": "obs"},
				},
			},
		})).To(Succeed())

		r := &PrometheusClusterSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: setName}})
		Expect(err).NotTo(HaveOccurred())

		var got observabilityv1.PrometheusClusterSet
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: setName}, &got)).To(Succeed())
		Expect(got.Status.MemberCount).To(Equal(int32(1)))
		Expect(got.Status.Members).To(HaveLen(1))
		Expect(got.Status.Members[0].Name).To(Equal("alpha"))
	})

	It("matches everything when selector is nil", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "gamma", Namespace: nsA},
			Spec:       observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
		})).To(Succeed())

		r := &PrometheusClusterSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: setName}})
		Expect(err).NotTo(HaveOccurred())

		var got observabilityv1.PrometheusClusterSet
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: setName}, &got)).To(Succeed())
		Expect(got.Status.MemberCount).To(BeNumerically(">=", 1))
	})
})
