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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrometheusClusterSetSpec defines the desired state of a PrometheusClusterSet —
// a cluster-scoped grouping of PrometheusCluster resources matched by label
// across namespaces. The Set itself does not create Prometheus instances; it
// observes membership and (optionally) carries a default backup template that
// member clusters can inherit.
type PrometheusClusterSetSpec struct {
	// ClusterSelector picks PrometheusCluster resources by labels. An empty
	// selector matches every PrometheusCluster in every namespace.
	// +optional
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// NamespaceSelector restricts which namespaces are considered. Empty
	// means all namespaces.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// BackupTemplate is a default backup spec applied as a fallback for
	// matched PrometheusCluster resources whose own spec.backup.enabled is
	// false. Member CRs always win on any field they set.
	// +optional
	BackupTemplate *S3BackupSpec `json:"backupTemplate,omitempty"`
}

// PrometheusClusterSetStatus defines the observed state of PrometheusClusterSet.
type PrometheusClusterSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the PrometheusClusterSet resource.
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

	// MemberCount is the number of PrometheusCluster resources currently
	// matched by the selectors.
	// +optional
	MemberCount int32 `json:"memberCount,omitempty"`

	// PhaseCount breaks the membership down by lifecycle phase.
	// +optional
	PhaseCount map[string]int32 `json:"phaseCount,omitempty"`

	// Members is the namespaced names of matched PrometheusCluster
	// resources, sorted for stability.
	// +optional
	Members []SetMember `json:"members,omitempty"`
}

// SetMember is one PrometheusCluster matched by a Set.
type SetMember struct {
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Phase     ClusterPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// PrometheusClusterSet is the Schema for the prometheusclustersets API
type PrometheusClusterSet struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PrometheusClusterSet
	// +required
	Spec PrometheusClusterSetSpec `json:"spec"`

	// status defines the observed state of PrometheusClusterSet
	// +optional
	Status PrometheusClusterSetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PrometheusClusterSetList contains a list of PrometheusClusterSet
type PrometheusClusterSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PrometheusClusterSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrometheusClusterSet{}, &PrometheusClusterSetList{})
}
