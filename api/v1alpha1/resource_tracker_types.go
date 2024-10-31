// api/v1alpha1/resource_tracker_types.go

// +kubebuilder:resource:path=resourcetrackers,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:object:generate=true
// +kubebuilder:resource:path=resourcetrackers,scope=Namespaced,categories={monitoring,ddukbg}
// +groupName=ddukbg.k8s
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceTrackerSpec defines the desired state of ResourceTracker
type ResourceTrackerSpec struct {
	// 1단계에서는 Deployment와 StatefulSet만 우선 지원
	Target ResourceTarget `json:"target"`
	Notify NotifyConfig   `json:"notify"`
}

// ResourceTarget defines the target resource to monitor
type ResourceTarget struct {
	// +kubebuilder:validation:Enum=Deployment;StatefulSet
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// NotifyConfig defines notification configuration
type NotifyConfig struct {
	Slack       string `json:"slack,omitempty"`
	Email       string `json:"email,omitempty"`
	RetryCount  int    `json:"retryCount,omitempty"`
	AlertOnFail bool   `json:"alertOnFail,omitempty"`
}

// ResourceState tracks the current state of the resource
type ResourceState struct {
	ImageState    ImageState `json:"imageState,omitempty"`
	ReadyReplicas int32      `json:"readyReplicas,omitempty"`
	TotalReplicas int32      `json:"totalReplicas,omitempty"`
}

// ImageState tracks image information
type ImageState struct {
	Tag       string `json:"tag,omitempty"`
	Digest    string `json:"digest,omitempty"`
	FullImage string `json:"fullImage,omitempty"`
}

// ResourceTrackerStatus defines the observed state of ResourceTracker
type ResourceTrackerStatus struct {
	Ready        bool          `json:"ready,omitempty"`
	LastUpdated  *metav1.Time  `json:"lastUpdated,omitempty"`
	CurrentState ResourceState `json:"currentState,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=resourcetrackers,scope=Namespaced,shortName=rt
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.target.kind"
// +kubebuilder:printcolumn:name="Target",type="string",JSONPath=".spec.target.name"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".spec.target.namespace"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ResourceTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceTrackerSpec   `json:"spec,omitempty"`
	Status ResourceTrackerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceTrackerList contains a list of ResourceTracker
type ResourceTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceTracker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceTracker{}, &ResourceTrackerList{})
}
