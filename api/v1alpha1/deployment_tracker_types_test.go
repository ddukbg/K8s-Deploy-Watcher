package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestDeploymentTrackerTypes(t *testing.T) {
	testTime := metav1.Time{Time: time.Now()}

	tests := []struct {
		name    string
		tracker *DeploymentTracker
	}{
		{
			name: "기본 설정",
			tracker: &DeploymentTracker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tracker",
					Namespace: "default",
				},
				Spec: DeploymentTrackerSpec{
					DeploymentName: "test-deployment",
					Namespace:      "default",
					Notify: NotifyConfig{
						Slack: "https://hooks.slack.com/test",
						Email: "test@example.com",
					},
				},
				Status: DeploymentTrackerStatus{
					Ready:            true,
					LastUpdated:      &testTime,
					ObservedReplicas: 3,
					ReadyReplicas:    3,
					Message:          "Deployment is ready",
				},
			},
		},
		{
			name: "최소 설정",
			tracker: &DeploymentTracker{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minimal-tracker",
				},
				Spec: DeploymentTrackerSpec{
					DeploymentName: "minimal-deployment",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DeepCopy 테스트
			copied := tt.tracker.DeepCopy()
			assert.Equal(t, tt.tracker.Name, copied.Name)
			assert.Equal(t, tt.tracker.Namespace, copied.Namespace)
			assert.Equal(t, tt.tracker.Spec, copied.Spec)
			assert.Equal(t, tt.tracker.Status, copied.Status)

			// DeepCopyObject 테스트
			copiedObj := tt.tracker.DeepCopyObject()
			copiedTracker, ok := copiedObj.(*DeploymentTracker)
			assert.True(t, ok)
			assert.Equal(t, tt.tracker.Name, copiedTracker.Name)

			// List DeepCopy 테스트
			list := &DeploymentTrackerList{
				Items: []DeploymentTracker{*tt.tracker},
			}
			copiedList := list.DeepCopy()
			assert.Equal(t, 1, len(copiedList.Items))
			assert.Equal(t, tt.tracker.Name, copiedList.Items[0].Name)
		})
	}
}

func TestNotifyConfig(t *testing.T) {
	tests := []struct {
		name   string
		config NotifyConfig
		valid  bool
	}{
		{
			name: "모든 알림 설정",
			config: NotifyConfig{
				Slack:       "https://hooks.slack.com/test",
				Email:       "test@example.com",
				RetryCount:  3,
				AlertOnFail: true,
			},
			valid: true,
		},
		{
			name: "Slack만 설정",
			config: NotifyConfig{
				Slack: "https://hooks.slack.com/test",
			},
			valid: true,
		},
		{
			name: "Email만 설정",
			config: NotifyConfig{
				Email: "test@example.com",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, len(tt.config.Slack) > 0 || len(tt.config.Email) > 0)
		})
	}
}
