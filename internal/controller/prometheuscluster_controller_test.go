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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

var _ = Describe("PrometheusCluster Controller", func() {
	const (
		resourceName = "test-resource"
		namespace    = "default"
	)

	ctx := context.Background()
	typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: namespace}

	reconcilerFor := func() *PrometheusClusterReconciler {
		return &PrometheusClusterReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
	}

	reconcileUntilStable := func(r *PrometheusClusterReconciler) {
		for i := 0; i < 5; i++ {
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			if !res.Requeue {
				return
			}
		}
	}

	AfterEach(func() {
		resource := &observabilityv1.PrometheusCluster{}
		if err := k8sClient.Get(ctx, typeNamespacedName, resource); err == nil {
			controllerutil.RemoveFinalizer(resource, finalizerName)
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		}
	})

	It("creates StatefulSet, Service, ConfigMap and sets Provisioning phase", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec: observabilityv1.PrometheusClusterSpec{
				Replicas: 2,
			},
		})).To(Succeed())

		r := reconcilerFor()
		reconcileUntilStable(r)

		By("creating a headless Service")
		var svc corev1.Service
		Expect(k8sClient.Get(ctx, typeNamespacedName, &svc)).To(Succeed())
		Expect(svc.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))

		By("creating a ConfigMap with prometheus.yml")
		var cm corev1.ConfigMap
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-config", Namespace: namespace}, &cm)).To(Succeed())
		Expect(cm.Data).To(HaveKey("prometheus.yml"))

		By("creating a StatefulSet mounting the config")
		var sts appsv1.StatefulSet
		Expect(k8sClient.Get(ctx, typeNamespacedName, &sts)).To(Succeed())
		Expect(*sts.Spec.Replicas).To(Equal(int32(2)))
		var cfgVolumeFound bool
		for _, v := range sts.Spec.Template.Spec.Volumes {
			if v.ConfigMap != nil && v.ConfigMap.Name == resourceName+"-config" {
				cfgVolumeFound = true
			}
		}
		Expect(cfgVolumeFound).To(BeTrue(), "config volume should be present")

		By("setting phase to Provisioning and adding the finalizer")
		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, typeNamespacedName, &pc)).To(Succeed())
		Expect(pc.Status.Phase).To(Equal(observabilityv1.PhaseProvisioning))
		Expect(controllerutil.ContainsFinalizer(&pc, finalizerName)).To(BeTrue())
	})

	It("adds --web.enable-admin-api when backup is enabled", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec: observabilityv1.PrometheusClusterSpec{
				Replicas: 1,
				Backup:   observabilityv1.S3BackupSpec{Enabled: true, Bucket: "b"},
			},
		})).To(Succeed())

		r := reconcilerFor()
		reconcileUntilStable(r)

		var sts appsv1.StatefulSet
		Expect(k8sClient.Get(ctx, typeNamespacedName, &sts)).To(Succeed())
		Expect(sts.Spec.Template.Spec.Containers[0].Args).To(ContainElement("--web.enable-admin-api"))
	})

	It("scales the StatefulSet when spec.replicas changes", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec:       observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())

		r := reconcilerFor()
		reconcileUntilStable(r)

		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, typeNamespacedName, &pc)).To(Succeed())
		pc.Spec.Replicas = 3
		Expect(k8sClient.Update(ctx, &pc)).To(Succeed())

		reconcileUntilStable(r)

		var sts appsv1.StatefulSet
		Expect(k8sClient.Get(ctx, typeNamespacedName, &sts)).To(Succeed())
		Expect(*sts.Spec.Replicas).To(Equal(int32(3)))

		Expect(k8sClient.Get(ctx, typeNamespacedName, &pc)).To(Succeed())
		Expect(pc.Status.Phase).To(Equal(observabilityv1.PhaseScaling))
	})

	It("removes the finalizer on delete", func() {
		Expect(k8sClient.Create(ctx, &observabilityv1.PrometheusCluster{
			ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespace},
			Spec:       observabilityv1.PrometheusClusterSpec{Replicas: 1},
		})).To(Succeed())

		r := reconcilerFor()
		reconcileUntilStable(r)

		var pc observabilityv1.PrometheusCluster
		Expect(k8sClient.Get(ctx, typeNamespacedName, &pc)).To(Succeed())
		Expect(k8sClient.Delete(ctx, &pc)).To(Succeed())

		reconcileUntilStable(r)

		err := k8sClient.Get(ctx, typeNamespacedName, &pc)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "CR should be fully deleted after finalizer runs")
	})
})
