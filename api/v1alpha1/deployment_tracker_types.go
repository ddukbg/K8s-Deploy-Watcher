package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentTrackerSpec defines the desired state of DeploymentTracker
type DeploymentTrackerSpec struct {
	DeploymentName string `json:"deploymentName,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	Notify         Notify `json:"notify"`
}

// Notify defines notification details (Slack or Email)
type Notify struct {
	Slack string `json:"slack,omitempty"`
	Email string `json:"email,omitempty"`
}

// DeploymentTrackerStatus defines the observed state of DeploymentTracker
type DeploymentTrackerStatus struct {
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentTracker is the Schema for the deploymenttrackers API
type DeploymentTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentTrackerSpec   `json:"spec,omitempty"`
	Status DeploymentTrackerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentTrackerList contains a list of DeploymentTracker
type DeploymentTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentTracker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentTracker{}, &DeploymentTrackerList{})
}
