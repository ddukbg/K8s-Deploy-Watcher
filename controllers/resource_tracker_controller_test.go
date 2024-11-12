// controllers/resource_tracker_controller_test.go

package controllers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	ddukbgv1alpha1 "k8s-deploy-watcher/api/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme" // 이 줄 추가
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var receivedMessages = make(chan string, 10)

func init() {
	sendSlackNotification = func(webhookURL string, message string) error {
		receivedMessages <- message
		return nil
	}
}

func TestReconcileResourceTracker(t *testing.T) {
	// 전체 테스트 타임아웃 설정 제거
	// testTimeout := time.After(time.Second * 10) // 이 줄 제거

	// 테스트 케이스 정의
	tests := []struct {
		name          string
		kind          string
		resourceName  string
		replicas      int32
		readyReplicas int32
		initialImage  string
		updatedImage  string
		expectReady   bool
		expectError   bool
	}{
		{
			name:          "정상적인 Deployment 추적",
			kind:          "Deployment",
			resourceName:  "test-app",
			replicas:      3,
			readyReplicas: 3,
			initialImage:  "nginx:1.19",
			updatedImage:  "nginx:1.20",
			expectReady:   true,
			expectError:   false,
		},
		{
			name:          "배포 진행 중",
			kind:          "Deployment",
			resourceName:  "test-app",
			replicas:      3,
			readyReplicas: 1,
			initialImage:  "nginx:1.19",
			updatedImage:  "",
			expectReady:   false,
			expectError:   false,
		},
		{
			name:          "정상적인 StatefulSet 추적",
			kind:          "StatefulSet",
			resourceName:  "test-mysql",
			replicas:      3,
			readyReplicas: 3,
			initialImage:  "mysql:5.7",
			updatedImage:  "mysql:8.0",
			expectReady:   true,
			expectError:   false,
		},
		{
			name:          "StatefulSet 배포 진행 중",
			kind:          "StatefulSet",
			resourceName:  "test-mysql",
			replicas:      3,
			readyReplicas: 1,
			initialImage:  "mysql:5.7",
			updatedImage:  "",
			expectReady:   false,
			expectError:   false,
		},
		{
			name:          "지원하지 않는 리소스 종류",
			kind:          "InvalidKind",
			resourceName:  "test-invalid",
			replicas:      1,
			readyReplicas: 1,
			expectReady:   false,
			expectError:   true,
		},
	}

	// Slack 알림 모의 설정
	receivedMessages := make(chan string, 1)
	originalSendSlack := sendSlackNotification
	defer func() { sendSlackNotification = originalSendSlack }()

	sendSlackNotification = func(webhookURL, message string) error {
		receivedMessages <- message
		return nil
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			// ResourceTracker 초기 상태 설정
			tracker := &ddukbgv1alpha1.ResourceTracker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tracker",
					Namespace: "default",
				},
				Spec: ddukbgv1alpha1.ResourceTrackerSpec{
					Target: ddukbgv1alpha1.ResourceTarget{
						Kind:      tt.kind,
						Name:      tt.resourceName,
						Namespace: "default",
					},
					Notify: ddukbgv1alpha1.NotifyConfig{
						Slack: "https://hooks.slack.com/test",
					},
				},
				Status: ddukbgv1alpha1.ResourceTrackerStatus{
					ResourceStatus:   make(map[string]bool),
					GenerationStatus: make(map[string]string),
				},
			}

			// 클라이언트 설정
			scheme := runtime.NewScheme()
			_ = clientgoscheme.AddToScheme(scheme)
			_ = ddukbgv1alpha1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			clientBuilder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&ddukbgv1alpha1.ResourceTracker{})

			// 리소스 생성
			if tt.kind == "Deployment" {
				deploy := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:       tt.resourceName,
						Namespace:  "default",
						Generation: 1,
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &tt.replicas,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": tt.resourceName},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": tt.resourceName},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "app",
										Image: tt.initialImage,
									},
								},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:           tt.replicas,
						ReadyReplicas:      tt.readyReplicas,
						UpdatedReplicas:    tt.readyReplicas,
						AvailableReplicas:  tt.readyReplicas,
						ObservedGeneration: 1,
					},
				}
				clientBuilder = clientBuilder.WithObjects(deploy)
			} else if tt.kind == "StatefulSet" {
				sts := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:       tt.resourceName,
						Namespace:  "default",
						Generation: 1,
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &tt.replicas,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": tt.resourceName},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": tt.resourceName},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "app",
										Image: tt.initialImage,
									},
								},
							},
						},
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:           tt.replicas,
						ReadyReplicas:      tt.readyReplicas,
						UpdatedReplicas:    tt.readyReplicas,
						ObservedGeneration: 1,
					},
				}
				clientBuilder = clientBuilder.WithObjects(sts)
			}

			// ResourceTracker 추가
			clientBuilder = clientBuilder.WithObjects(tracker)
			client := clientBuilder.Build()

			// Recorder 설정
			recorder := record.NewFakeRecorder(10)

			r := &ResourceTrackerReconciler{
				Client:   client,
				Scheme:   scheme,
				Recorder: recorder,
			}

			// Reconcile 실행
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tracker.Name,
					Namespace: tracker.Namespace,
				},
			}

			result, err := r.Reconcile(ctx, req)

			if tt.kind == "InvalidKind" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported resource kind")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, time.Second*30, result.RequeueAfter)

			// 상태가 업데이트될 때까지 대기
			time.Sleep(time.Millisecond * 100)

			if tt.expectReady {
				// 이벤트 확인
				select {
				case event := <-recorder.Events:
					t.Logf("Received event: %s", event)
					expectedEvent := ""
					switch tt.kind {
					case "Deployment":
						expectedEvent = "DeploymentReady"
					case "StatefulSet":
						expectedEvent = "StatefulSetReady"
					case "Pod":
						expectedEvent = "PodReady"
					}
					assert.Contains(t, event, expectedEvent)
				case <-time.After(time.Second):
					t.Error("Expected event not received")
				}

				// Slack 알림 확인
				select {
				case msg := <-receivedMessages:
					t.Logf("Received Slack message: %s", msg)
					expectedMsg := formatSlackMessage(tt.kind, "default", tt.resourceName,
						tt.readyReplicas, tt.replicas)
					assert.Equal(t, expectedMsg, msg)
				case <-time.After(time.Second):
					t.Error("Expected Slack notification not received")
				}
			}
		})
	}
}

func TestReconcileEvents(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = ddukbgv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Slack 알림 테스트를 위한 채널
	receivedMessages := make(chan string, 1)
	originalSendSlackNotification := sendSlackNotification
	defer func() {
		sendSlackNotification = originalSendSlackNotification
	}()

	testCases := []struct {
		name         string
		kind         string
		resourceName string
		resource     runtime.Object
	}{
		{
			name:         "Deployment 이벤트",
			kind:         "Deployment",
			resourceName: "test-app",
			resource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-app",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas:      3,
					UpdatedReplicas:    3,
					AvailableReplicas:  3,
					ObservedGeneration: 1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// ResourceTracker 생성 시 초기 상태 설정
			tracker := &ddukbgv1alpha1.ResourceTracker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tracker",
					Namespace: "default",
				},
				Spec: ddukbgv1alpha1.ResourceTrackerSpec{
					Target: ddukbgv1alpha1.ResourceTarget{
						Kind:      tc.kind,
						Name:      tc.resourceName,
						Namespace: "default",
					},
					Notify: ddukbgv1alpha1.NotifyConfig{
						Slack: "https://hooks.slack.com/test",
					},
				},
				Status: ddukbgv1alpha1.ResourceTrackerStatus{
					Ready:   false,
					Message: "",
				},
			}

			// Mock Slack 알림 함수 설정
			sendSlackNotification = func(webhookURL string, message string) error {
				receivedMessages <- message
				return nil
			}

			// Fake 클라이언트 및 Recorder 설정
			recorder := record.NewFakeRecorder(10)
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tracker).
				WithRuntimeObjects(tc.resource).
				WithStatusSubresource(&ddukbgv1alpha1.ResourceTracker{}, &appsv1.Deployment{}, &appsv1.StatefulSet{}).
				Build()

			r := &ResourceTrackerReconciler{
				Client:   client,
				Scheme:   scheme,
				Recorder: recorder,
			}

			// Reconcile 실행
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tracker.Name,
					Namespace: tracker.Namespace,
				},
			}

			// 첫 번째 Reconcile - 초기 상태 설정
			_, err := r.Reconcile(ctx, req)
			require.NoError(t, err)

			// 상태 변경을 위해 리소스 업데이트
			updatedTracker := &ddukbgv1alpha1.ResourceTracker{}
			err = client.Get(ctx, req.NamespacedName, updatedTracker)
			require.NoError(t, err)

			// 두 번째 Reconcile - 상태 변경 감지
			_, err = r.Reconcile(ctx, req)
			require.NoError(t, err)

			// 이벤트 확인 (여러 번 시도)
			t.Log("Waiting for event...")
			eventReceived := false
			timeout := time.After(time.Second * 2)

			for !eventReceived {
				select {
				case event := <-recorder.Events:
					t.Logf("Received event: %s", event)
					if strings.Contains(event, "DeploymentReady") {
						eventReceived = true
						break
					}
				case <-timeout:
					t.Error("Expected event not received")
					return
				}
			}

			// 상태 확인
			key := fmt.Sprintf("%s/%s", "default", "test-app")
			assert.True(t, updatedTracker.Status.ResourceStatus[key], "Resource should be marked as ready")

			// Slack 알림 확인
			t.Log("Waiting for Slack notification...")
			select {
			case msg := <-receivedMessages:
				t.Logf("Received Slack message: %s", msg)
				expectedMsg := formatSlackMessage("Deployment", "default", "test-app",
					int32(3), int32(3))
				assert.Equal(t, expectedMsg, msg)
			case <-time.After(time.Second * 2):
				t.Error("Expected Slack notification not received")
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
