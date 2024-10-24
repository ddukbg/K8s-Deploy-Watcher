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

// DeploymentTrackerStatus에 더 자세한 상태 정보 추가 필요
type DeploymentTrackerStatus struct {
    Ready bool `json:"ready,omitempty"`
    // 추가할 필드들:
    LastUpdated       metav1.Time `json:"lastUpdated,omitempty"`
    ObservedReplicas int32       `json:"observedReplicas,omitempty"`
    ReadyReplicas    int32       `json:"readyReplicas,omitempty"`
    Message          string      `json:"message,omitempty"`
}

// Notify 구조체에 알림 설정 추가
type Notify struct {
    Slack       string `json:"slack,omitempty"`
    Email       string `json:"email,omitempty"`
    // 추가할 필드들:
    RetryCount  int    `json:"retryCount,omitempty"`
    AlertOnFail bool   `json:"alertOnFail,omitempty"`
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
