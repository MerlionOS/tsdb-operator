/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterPhase is the high-level lifecycle state of a PrometheusCluster.
type ClusterPhase string

const (
	PhaseProvisioning ClusterPhase = "Provisioning"
	PhaseActive       ClusterPhase = "Active"
	PhaseScaling      ClusterPhase = "Scaling"
	PhaseFailed       ClusterPhase = "Failed"
)

// S3BackupSpec describes where backups are shipped.
type S3BackupSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	Bucket string `json:"bucket,omitempty"`
	// +optional
	Region string `json:"region,omitempty"`
	// Endpoint overrides the S3 endpoint (e.g. for MinIO).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
	// Prefix under which snapshot objects are stored.
	// +optional
	Prefix string `json:"prefix,omitempty"`
	// Cron expression for scheduled snapshots. Standard 5-field cron
	// syntax; deeper validation is done by the webhook.
	// +kubebuilder:validation:MinLength=1
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// CredentialsSecretRef references a Secret with AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY.
	// +optional
	CredentialsSecretRef *corev1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`
}

// RemoteWriteSpec declares a Prometheus remote_write target. Fields mirror
// the subset of the upstream Prometheus remote_write schema we render into
// the generated prometheus.yml.
type RemoteWriteSpec struct {
	// URL is the remote endpoint to ship samples to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Name is an optional label for the queue (used in Prometheus metrics).
	// +optional
	Name string `json:"name,omitempty"`

	// BasicAuthSecretRef references a Secret with keys "username" and "password".
	// +optional
	BasicAuthSecretRef *corev1.LocalObjectReference `json:"basicAuthSecretRef,omitempty"`

	// BearerTokenSecretRef references a Secret with key "token".
	// +optional
	BearerTokenSecretRef *corev1.LocalObjectReference `json:"bearerTokenSecretRef,omitempty"`
}

// ThanosSpec opts the cluster into a Thanos sidecar. The sidecar reads the
// same TSDB data volume as Prometheus and ships 2h blocks to object storage.
// Pair with Thanos Query elsewhere for a global query view.
type ThanosSpec struct {
	// Enabled turns the sidecar on.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image is the Thanos container image.
	// +kubebuilder:default="quay.io/thanos/thanos:v0.36.1"
	// +optional
	Image string `json:"image,omitempty"`

	// ObjectStorageConfigSecretRef references a Secret with an
	// "objstore.yml" key in the Thanos objstore format. Sidecar reads it
	// via --objstore.config-file.
	// +optional
	ObjectStorageConfigSecretRef *corev1.LocalObjectReference `json:"objectStorageConfigSecretRef,omitempty"`
}

// StorageSpec describes the PVC used by each replica.
type StorageSpec struct {
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// Size of the PVC, e.g. "50Gi".
	// +kubebuilder:default="20Gi"
	Size resource.Quantity `json:"size,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrometheusClusterSpec defines the desired state of PrometheusCluster
type PrometheusClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Replicas is the desired number of Prometheus replicas.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the Prometheus container image.
	// +kubebuilder:default="prom/prometheus:v2.53.0"
	Image string `json:"image,omitempty"`

	// Retention period for local TSDB data, e.g. "15d".
	// Accepts Prometheus duration syntax: [0-9]+(ms|s|m|h|d|w|y).
	// +kubebuilder:default="15d"
	// +kubebuilder:validation:Pattern=`^[0-9]+(ms|s|m|h|d|w|y)$`
	Retention string `json:"retention,omitempty"`

	// Storage describes the per-replica PVC.
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Resources is the container resource requests/limits.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Backup configuration.
	// +optional
	Backup S3BackupSpec `json:"backup,omitempty"`

	// RemoteWrite endpoints the managed Prometheus should stream samples to.
	// +optional
	RemoteWrite []RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// Thanos optionally attaches a Thanos sidecar to each replica.
	// +optional
	Thanos ThanosSpec `json:"thanos,omitempty"`

	// AdditionalScrapeConfigs supplies user-side scrape configuration
	// merged into the generated prometheus.yml via the native
	// scrape_config_files mechanism (Prometheus 2.43+).
	// Exactly one of `inline` or `secretRef` may be set.
	// +optional
	AdditionalScrapeConfigs *AdditionalScrapeConfigs `json:"additionalScrapeConfigs,omitempty"`
}

// AdditionalScrapeConfigs carries either inline YAML or a reference to a
// Secret that holds the scrape configuration. Mutually exclusive — the
// admission webhook rejects setting both.
type AdditionalScrapeConfigs struct {
	// Inline is a top-level YAML list of scrape entries, e.g.
	//
	//   - job_name: my-app
	//     static_configs:
	//       - targets: [my-app:8080]
	//
	// The operator wraps it under a `scrape_configs:` key in the
	// generated ConfigMap. Use this when the config is small enough to
	// live in the CR.
	// +optional
	Inline string `json:"inline,omitempty"`

	// SecretRef references a Secret containing a fully-formed Prometheus
	// scrape config file (must already include the top-level
	// `scrape_configs:` key). The Secret is mounted into the Prometheus
	// container at /etc/prometheus/extra-secret/<key>. Use this for
	// configs too large or sensitive to inline.
	// +optional
	SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`
}

// PrometheusClusterStatus defines the observed state of PrometheusCluster.
type PrometheusClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the PrometheusCluster resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is the high-level lifecycle state.
	// +kubebuilder:validation:Enum=Provisioning;Active;Scaling;Failed
	// +optional
	Phase ClusterPhase `json:"phase,omitempty"`

	// ReadyReplicas is the number of replicas reporting Ready.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// LastBackupTime is the timestamp of the most recent successful backup.
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// LastFailoverTime is the timestamp of the most recent failover event.
	// +optional
	LastFailoverTime *metav1.Time `json:"lastFailoverTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PrometheusCluster is the Schema for the prometheusclusters API
type PrometheusCluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PrometheusCluster
	// +required
	Spec PrometheusClusterSpec `json:"spec"`

	// status defines the observed state of PrometheusCluster
	// +optional
	Status PrometheusClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PrometheusClusterList contains a list of PrometheusCluster
type PrometheusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PrometheusCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrometheusCluster{}, &PrometheusClusterList{})
}
