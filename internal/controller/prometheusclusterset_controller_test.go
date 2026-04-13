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

	It("overlays backupTemplate onto members without their own backup config", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "inherits", Namespace: nsA,
				Labels: map[string]string{"team": "obs"},
			},
			Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())

		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
			Spec: observabilityv1.PrometheusClusterSetSpec{
				ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "obs"}},
				BackupTemplate: &observabilityv1.S3BackupSpec{
					Bucket:   "set-bucket",
					Schedule: "0 */6 * * *",
				},
			},
		})).To(Succeed())

		r := &PrometheusClusterSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: setName}})
		Expect(err).NotTo(HaveOccurred())

		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "inherits", Namespace: nsA}, &pc)).To(Succeed())
		Expect(pc.Spec.Backup.Enabled).To(BeTrue())
		Expect(pc.Spec.Backup.Bucket).To(Equal("set-bucket"))
		Expect(pc.Annotations[ManagedByAnnotation]).To(Equal(setName))
	})

	It("leaves members alone when they carry the opt-out annotation", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "optout", Namespace: nsA,
				Labels:      map[string]string{"team": "obs"},
				Annotations: map[string]string{OptOutAnnotation: "true"},
			},
			Spec: observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
			Spec: observabilityv1.PrometheusClusterSetSpec{
				ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "obs"}},
				BackupTemplate:  &observabilityv1.S3BackupSpec{Bucket: "set-bucket"},
			},
		})).To(Succeed())

		r := &PrometheusClusterSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: setName}})
		Expect(err).NotTo(HaveOccurred())

		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "optout", Namespace: nsA}, &pc)).To(Succeed())
		Expect(pc.Spec.Backup.Enabled).To(BeFalse())
		Expect(pc.Annotations).NotTo(HaveKey(ManagedByAnnotation))
	})

	It("leaves members alone when they already have backup enabled", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "owner", Namespace: nsA,
				Labels: map[string]string{"team": "obs"},
			},
			Spec: observabilityv1.PrometheusClusterSpec{
				Replicas: 1,
				Backup:   observabilityv1.S3BackupSpec{Enabled: true, Bucket: "own-bucket"},
			},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusClusterSet{
			ObjectMeta: metav1.ObjectMeta{Name: setName},
			Spec: observabilityv1.PrometheusClusterSetSpec{
				ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "obs"}},
				BackupTemplate:  &observabilityv1.S3BackupSpec{Bucket: "set-bucket"},
			},
		})).To(Succeed())

		r := &PrometheusClusterSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: setName}})
		Expect(err).NotTo(HaveOccurred())

		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "owner", Namespace: nsA}, &pc)).To(Succeed())
		Expect(pc.Spec.Backup.Bucket).To(Equal("own-bucket"))
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
