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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// PrometheusClusterReconciler reconciles a PrometheusCluster object.
type PrometheusClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *PrometheusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pc observabilityv1.PrometheusCluster
	if err := r.Get(ctx, req.NamespacedName, &pc); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get PrometheusCluster: %w", err)
	}

	if !pc.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileHeadlessService(ctx, &pc); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile service: %w", err)
	}

	current := &appsv1.StatefulSet{}
	desired := r.buildStatefulSet(&pc)
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case apierrors.IsNotFound(err):
		if err := controllerutil.SetControllerReference(&pc, desired, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("set owner ref: %w", err)
		}
		if err := r.Create(ctx, desired); err != nil {
			return ctrl.Result{}, fmt.Errorf("create statefulset: %w", err)
		}
		log.Info("created statefulset", "name", desired.Name)
		return r.updatePhase(ctx, &pc, observabilityv1.PhaseProvisioning, 0)
	case err != nil:
		return ctrl.Result{}, fmt.Errorf("get statefulset: %w", err)
	}

	phase := observabilityv1.PhaseActive
	if *current.Spec.Replicas != pc.Spec.Replicas {
		current.Spec.Replicas = &pc.Spec.Replicas
		current.Spec.Template = desired.Spec.Template
		if err := r.Update(ctx, current); err != nil {
			return ctrl.Result{}, fmt.Errorf("update statefulset: %w", err)
		}
		phase = observabilityv1.PhaseScaling
	} else if current.Status.ReadyReplicas < pc.Spec.Replicas {
		phase = observabilityv1.PhaseProvisioning
	}

	return r.updatePhase(ctx, &pc, phase, current.Status.ReadyReplicas)
}

func (r *PrometheusClusterReconciler) updatePhase(ctx context.Context, pc *observabilityv1.PrometheusCluster, phase observabilityv1.ClusterPhase, ready int32) (ctrl.Result, error) {
	pc.Status.Phase = phase
	pc.Status.ReadyReplicas = ready
	if err := r.Status().Update(ctx, pc); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *PrometheusClusterReconciler) reconcileHeadlessService(ctx context.Context, pc *observabilityv1.PrometheusCluster) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: pc.Name, Namespace: pc.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec.ClusterIP = corev1.ClusterIPNone
		svc.Spec.Selector = map[string]string{"app.kubernetes.io/instance": pc.Name}
		svc.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       9090,
			TargetPort: intstr.FromInt(9090),
		}}
		return controllerutil.SetControllerReference(pc, svc, r.Scheme)
	})
	return err
}

func (r *PrometheusClusterReconciler) buildStatefulSet(pc *observabilityv1.PrometheusCluster) *appsv1.StatefulSet {
	labels := map[string]string{
		"app.kubernetes.io/name":     "prometheus",
		"app.kubernetes.io/instance": pc.Name,
	}
	size := pc.Spec.Storage.Size
	if size.IsZero() {
		size = resource.MustParse("20Gi")
	}
	image := pc.Spec.Image
	if image == "" {
		image = "prom/prometheus:v2.53.0"
	}
	retention := pc.Spec.Retention
	if retention == "" {
		retention = "15d"
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pc.Name,
			Namespace: pc.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: pc.Name,
			Replicas:    &pc.Spec.Replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "prometheus",
						Image: image,
						Args: []string{
							"--config.file=/etc/prometheus/prometheus.yml",
							"--storage.tsdb.path=/prometheus",
							fmt.Sprintf("--storage.tsdb.retention.time=%s", retention),
							"--web.enable-lifecycle",
						},
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 9090}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
								Path: "/-/ready", Port: intstr.FromInt(9090),
							}},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
								Path: "/-/healthy", Port: intstr.FromInt(9090),
							}},
						},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/prometheus"}},
						Resources:    pc.Spec.Resources,
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: pc.Spec.Storage.StorageClassName,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: size},
					},
				},
			}},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.PrometheusCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Named("prometheuscluster").
		Complete(r)
}
