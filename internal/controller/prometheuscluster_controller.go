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
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
	"github.com/MerlionOS/tsdb-operator/internal/metrics"
)

const finalizerName = "observability.merlionos.org/finalizer"

// scrapeConfig is the base scrape block. The global block is rendered
// separately by renderConfig because Thanos adds external_labels to it.
const scrapeConfig = `scrape_configs:
  - job_name: prometheus
    static_configs:
      - targets: ['localhost:9090']
`

// BackupRegistrar is the subset of the backup scheduler the reconciler needs.
// Decouples the controller package from internal/backup.
type BackupRegistrar interface {
	Register(ctx context.Context, pc *observabilityv1.PrometheusCluster) error
}

// PrometheusClusterReconciler reconciles a PrometheusCluster object.
type PrometheusClusterReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	BackupSchedule BackupRegistrar // optional; set when backup is enabled

	// HTTP is the client used for /-/reload calls. Defaulted lazily to
	// http.DefaultClient when nil; tests inject a fake transport.
	HTTP *http.Client
}

// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.merlionos.org,resources=prometheusclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
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
		return r.finalize(ctx, &pc)
	}

	if !controllerutil.ContainsFinalizer(&pc, finalizerName) {
		controllerutil.AddFinalizer(&pc, finalizerName)
		if err := r.Update(ctx, &pc); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if r.BackupSchedule != nil && pc.Spec.Backup.Enabled {
		if err := r.BackupSchedule.Register(ctx, &pc); err != nil {
			return ctrl.Result{}, fmt.Errorf("register backup: %w", err)
		}
	}

	cmChanged, err := r.reconcileConfigMap(ctx, &pc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile configmap: %w", err)
	}
	if cmChanged {
		// Fire-and-forget per-pod reload. Failures are logged inside.
		r.triggerReload(ctx, &pc)
	}
	if err := r.reconcileHeadlessService(ctx, &pc); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile service: %w", err)
	}

	current := &appsv1.StatefulSet{}
	desired := r.buildStatefulSet(&pc)
	err = r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
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

	replicasChanged := *current.Spec.Replicas != pc.Spec.Replicas
	templateChanged := !equality.Semantic.DeepEqual(current.Spec.Template.Spec, desired.Spec.Template.Spec)
	if replicasChanged || templateChanged {
		current.Spec.Replicas = &pc.Spec.Replicas
		current.Spec.Template = desired.Spec.Template
		if err := r.Update(ctx, current); err != nil {
			return ctrl.Result{}, fmt.Errorf("update statefulset: %w", err)
		}
	}

	phase := observabilityv1.PhaseActive
	switch {
	case replicasChanged:
		phase = observabilityv1.PhaseScaling
	case current.Status.ReadyReplicas < pc.Spec.Replicas:
		phase = observabilityv1.PhaseProvisioning
	}

	return r.updatePhase(ctx, &pc, phase, current.Status.ReadyReplicas)
}

// finalize handles cleanup on deletion. Owner references take care of the
// StatefulSet, Service, and ConfigMap; the finalizer is a hook for future
// work (e.g. a last-chance backup) and guarantees the audit trail sees it.
func (r *PrometheusClusterReconciler) finalize(ctx context.Context, pc *observabilityv1.PrometheusCluster) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(pc, finalizerName) {
		return ctrl.Result{}, nil
	}
	controllerutil.RemoveFinalizer(pc, finalizerName)
	if err := r.Update(ctx, pc); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
	}
	metrics.DeleteCluster(pc.Namespace, pc.Name)
	return ctrl.Result{}, nil
}

func (r *PrometheusClusterReconciler) updatePhase(ctx context.Context, pc *observabilityv1.PrometheusCluster, phase observabilityv1.ClusterPhase, ready int32) (ctrl.Result, error) {
	pc.Status.Phase = phase
	pc.Status.ReadyReplicas = ready
	if err := r.Status().Update(ctx, pc); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}
	metrics.SetPhase(pc.Namespace, pc.Name, string(phase))
	return ctrl.Result{}, nil
}

// reconcileConfigMap creates or updates the prometheus.yml ConfigMap and
// reports whether the existing object's content actually changed (so the
// caller can issue a /-/reload). First-time creation does not count as a
// change — pods will pick up the config on first start.
func (r *PrometheusClusterReconciler) reconcileConfigMap(ctx context.Context, pc *observabilityv1.PrometheusCluster) (bool, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: configMapName(pc), Namespace: pc.Namespace},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["prometheus.yml"] = renderConfig(pc)
		if pc.Spec.AdditionalScrapeConfigs != "" {
			cm.Data[additionalScrapeFile] = wrapScrapeConfigs(pc.Spec.AdditionalScrapeConfigs)
		} else {
			delete(cm.Data, additionalScrapeFile)
		}
		return controllerutil.SetControllerReference(pc, cm, r.Scheme)
	})
	if err != nil {
		return false, err
	}
	return op == controllerutil.OperationResultUpdated, nil
}

// triggerReload POSTs /-/reload to every Ready pod of the cluster. Best-
// effort: per-pod failures are logged but never propagated (the next
// reconcile will retry; pods restarting eventually pick up the config
// anyway).
func (r *PrometheusClusterReconciler) triggerReload(ctx context.Context, pc *observabilityv1.PrometheusCluster) {
	log := logf.FromContext(ctx).WithValues("cluster", pc.Name)
	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(pc.Namespace), client.MatchingLabels{
		"app.kubernetes.io/instance": pc.Name,
	}); err != nil {
		log.Error(err, "list pods for reload")
		return
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.PodIP == "" {
			continue
		}
		url := fmt.Sprintf("http://%s:9090/-/reload", pod.Status.PodIP)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Error(err, "reload failed", "pod", pod.Name)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			log.Info("reload returned non-2xx", "pod", pod.Name, "status", resp.StatusCode)
		}
	}
}

// additionalScrapeFile is the ConfigMap key (and basename inside the
// /etc/prometheus mount) for spec.additionalScrapeConfigs.
const additionalScrapeFile = "additional-scrape-configs.yml"

// wrapScrapeConfigs takes the user's bare YAML list of scrape entries and
// wraps it under a `scrape_configs:` key so the file matches what
// Prometheus 2.43+ scrape_config_files expects (a YAML object whose
// `scrape_configs` field is a list, not a bare list at the top level).
func wrapScrapeConfigs(s string) string {
	var b strings.Builder
	b.WriteString("scrape_configs:\n")
	for line := range strings.SplitSeq(strings.TrimRight(s, "\n"), "\n") {
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// renderConfig composes the prometheus.yml for a cluster: one global block
// (with Thanos external_labels when enabled), then the scrape config, then
// optional remote_write entries from spec.remoteWrite. Kept string-based
// for readability; upgrade to yaml.Marshal when the template gains
// conditionals.
func renderConfig(pc *observabilityv1.PrometheusCluster) string {
	var b strings.Builder
	b.WriteString("global:\n  scrape_interval: 30s\n  evaluation_interval: 30s\n")
	if pc.Spec.Thanos.Enabled {
		// Thanos requires uniquely-identifying external labels so samples
		// shipped from replicas don't collide in object storage. POD_NAME
		// expansion requires --enable-feature=expand-external-labels on
		// Prometheus.
		fmt.Fprintf(&b, "  external_labels:\n    cluster: %q\n    replica: ${POD_NAME:-unknown}\n", pc.Name)
	}
	b.WriteString(scrapeConfig)
	if pc.Spec.AdditionalScrapeConfigs != "" {
		// Prometheus 2.43+ scrape_config_files: load extra scrape entries from
		// the ConfigMap key mounted alongside prometheus.yml.
		fmt.Fprintf(&b, "scrape_config_files:\n  - /etc/prometheus/%s\n", additionalScrapeFile)
	}
	if len(pc.Spec.RemoteWrite) == 0 {
		return b.String()
	}
	b.WriteString("remote_write:\n")
	for _, rw := range pc.Spec.RemoteWrite {
		fmt.Fprintf(&b, "  - url: %q\n", rw.URL)
		if rw.Name != "" {
			fmt.Fprintf(&b, "    name: %q\n", rw.Name)
		}
		if rw.BasicAuthSecretRef != nil {
			fmt.Fprintf(&b, "    basic_auth:\n")
			fmt.Fprintf(&b, "      username_file: /etc/prometheus/secrets/%s/username\n", rw.BasicAuthSecretRef.Name)
			fmt.Fprintf(&b, "      password_file: /etc/prometheus/secrets/%s/password\n", rw.BasicAuthSecretRef.Name)
		}
		if rw.BearerTokenSecretRef != nil {
			fmt.Fprintf(&b, "    bearer_token_file: /etc/prometheus/secrets/%s/token\n", rw.BearerTokenSecretRef.Name)
		}
	}
	return b.String()
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

func configMapName(pc *observabilityv1.PrometheusCluster) string {
	return pc.Name + "-config"
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
	args := []string{
		"--config.file=/etc/prometheus/prometheus.yml",
		"--storage.tsdb.path=/prometheus",
		fmt.Sprintf("--storage.tsdb.retention.time=%s", retention),
		"--web.enable-lifecycle",
	}
	if pc.Spec.Backup.Enabled {
		args = append(args, "--web.enable-admin-api")
	}
	if pc.Spec.Thanos.Enabled {
		// Thanos sidecar refuses to start unless Prometheus-side compaction
		// is disabled (Thanos does its own compaction after shipping). It
		// also requires uniquely-identifying external_labels — rendered
		// into the ConfigMap with a ${POD_NAME} placeholder, which
		// expand-external-labels resolves at runtime.
		args = append(args,
			"--storage.tsdb.min-block-duration=2h",
			"--storage.tsdb.max-block-duration=2h",
			"--enable-feature=expand-external-labels",
		)
	}
	promEnv := []corev1.EnvVar{{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
		},
	}}

	containers := []corev1.Container{{
		Name:  "prometheus",
		Image: image,
		Args:  args,
		Env:   promEnv,
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
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: "/prometheus"},
			{Name: "config", MountPath: "/etc/prometheus"},
		},
		Resources: pc.Spec.Resources,
	}}

	volumes := []corev1.Volume{{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapName(pc)},
			},
		},
	}}

	if pc.Spec.Thanos.Enabled {
		sidecar, extraVolume := buildThanosSidecar(&pc.Spec.Thanos)
		containers = append(containers, sidecar)
		if extraVolume != nil {
			volumes = append(volumes, *extraVolume)
		}
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
					Containers: containers,
					Volumes:    volumes,
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

// buildThanosSidecar returns the sidecar container and an optional Volume
// for the objstore config Secret. Sidecar reads blocks from the shared
// /prometheus data volume and ships them to object storage.
func buildThanosSidecar(t *observabilityv1.ThanosSpec) (corev1.Container, *corev1.Volume) {
	image := t.Image
	if image == "" {
		image = "quay.io/thanos/thanos:v0.36.1"
	}
	args := []string{
		"sidecar",
		"--tsdb.path=/prometheus",
		"--prometheus.url=http://localhost:9090",
		"--http-address=0.0.0.0:10902",
		"--grpc-address=0.0.0.0:10901",
	}
	mounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/prometheus"},
	}
	var volume *corev1.Volume
	if t.ObjectStorageConfigSecretRef != nil {
		args = append(args, "--objstore.config-file=/etc/thanos/objstore/objstore.yml")
		mounts = append(mounts, corev1.VolumeMount{
			Name: "thanos-objstore", MountPath: "/etc/thanos/objstore", ReadOnly: true,
		})
		volume = &corev1.Volume{
			Name: "thanos-objstore",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: t.ObjectStorageConfigSecretRef.Name},
			},
		}
	}
	return corev1.Container{
		Name:  "thanos-sidecar",
		Image: image,
		Args:  args,
		Ports: []corev1.ContainerPort{
			{Name: "thanos-http", ContainerPort: 10902},
			{Name: "thanos-grpc", ContainerPort: 10901},
		},
		VolumeMounts: mounts,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
				Path: "/-/ready", Port: intstr.FromInt(10902),
			}},
		},
	}, volume
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.PrometheusCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Named("prometheuscluster").
		Complete(r)
}
