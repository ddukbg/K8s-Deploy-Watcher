// api/v1alpha1/deployment_tracker_types.go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentTrackerSpec defines the desired state of DeploymentTracker
type DeploymentTrackerSpec struct {
	DeploymentName string       `json:"deploymentName,omitempty"`
	Namespace      string       `json:"namespace,omitempty"`
	Notify         NotifyConfig `json:"notify"`
}

// NotifyConfig defines notification details
type NotifyConfig struct {
	Slack       string `json:"slack,omitempty"`
	Email       string `json:"email,omitempty"`
	RetryCount  int    `json:"retryCount,omitempty"`
	AlertOnFail bool   `json:"alertOnFail,omitempty"`
}

// DeploymentTrackerStatus defines the observed state of DeploymentTracker
type DeploymentTrackerStatus struct {
	Ready            bool         `json:"ready,omitempty"`
	LastUpdated      *metav1.Time `json:"lastUpdated,omitempty"`
	ObservedReplicas int32        `json:"observedReplicas,omitempty"`
	ReadyReplicas    int32        `json:"readyReplicas,omitempty"`
	Message          string       `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// DeploymentTracker is the Schema for the deploymenttrackers API
type DeploymentTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentTrackerSpec   `json:"spec,omitempty"`
	Status DeploymentTrackerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentTrackerList contains a list of DeploymentTracker
type DeploymentTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentTracker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentTracker{}, &DeploymentTrackerList{})
}

// DeepCopyInto is required for the DeploymentTrackerStatus
func (in *DeploymentTrackerStatus) DeepCopyInto(out *DeploymentTrackerStatus) {
	*out = *in
	if in.LastUpdated != nil {
		in, out := &in.LastUpdated, &out.LastUpdated
		*out = (*in).DeepCopy()
	}
}
