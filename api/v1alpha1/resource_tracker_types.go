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
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;Pod
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// +optional
	// Name is optional; if empty, all resources of the specified Kind in the Namespace will be monitored
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Required
	// Namespace to monitor resources in
	Namespace string `json:"namespace"`
}

// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.target.kind"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".spec.target.namespace"
// +kubebuilder:printcolumn:name="Resource",type="string",JSONPath=".spec.target.name"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NotifyConfig defines notification configuration
type NotifyConfig struct {
	Slack       string `json:"slack,omitempty"`
	Email       string `json:"email,omitempty"`
	RetryCount  int    `json:"retryCount,omitempty"`
	AlertOnFail bool   `json:"alertOnFail,omitempty"`
}

// ResourceState tracks the current state of the resource
type ResourceState struct {
	// Resource name
	Name string `json:"name"`

	// Current image information
	ImageState ImageState `json:"imageState,omitempty"`

	// Replicas status (for Deployment and StatefulSet)
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	TotalReplicas int32 `json:"totalReplicas,omitempty"`

	// Pod specific status
	PodPhase string `json:"podPhase,omitempty"`

	// Resource specific messages
	Message string `json:"message,omitempty"`

	// CurrentState 필드 추가
	CurrentImage  string `json:"currentImage,omitempty"`
	PreviousImage string `json:"previousImage,omitempty"`
}

// ImageState tracks image information
type ImageState struct {
	Tag       string `json:"tag,omitempty"`
	Digest    string `json:"digest,omitempty"`
	FullImage string `json:"fullImage,omitempty"`
}

// ResourceTrackerStatus defines the observed state of ResourceTracker
type ResourceTrackerStatus struct {
	// Overall ready status
	Ready bool `json:"ready,omitempty"`

	// Last time the status was updated
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Status of each resource being tracked
	ResourceStates []ResourceState `json:"resourceStates,omitempty"`

	// Overall status message
	Message string `json:"message,omitempty"`

	// CurrentState 필드 추가
	CurrentState ResourceState `json:"currentState,omitempty"`
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
