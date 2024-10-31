package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

func TestReconcile(t *testing.T) {
	// 1. 스키마 등록
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	replicas := int32(3)
	// 테스트 케이스들
	tests := []struct {
		name             string
		deploymentSpec   *appsv1.DeploymentSpec
		deploymentStatus *appsv1.DeploymentStatus
		expectedReady    bool
		expectError      bool
	}{
		{
			name: "성공적인 배포",
			deploymentSpec: &appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image:latest",
							},
						},
					},
				},
			},
			deploymentStatus: &appsv1.DeploymentStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
			expectedReady: true,
			expectError:   false,
		},
		{
			name: "진행 중인 배포",
			deploymentSpec: &appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image:latest",
							},
						},
					},
				},
			},
			deploymentStatus: &appsv1.DeploymentStatus{
				Replicas:      3,
				ReadyReplicas: 1,
			},
			expectedReady: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// 2. 테스트 객체 생성
			tracker := &v1alpha1.DeploymentTracker{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "ddukbg.k8s/v1alpha1",
					Kind:       "DeploymentTracker",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tracker",
					Namespace: "default",
				},
				Spec: v1alpha1.DeploymentTrackerSpec{
					DeploymentName: "test-deployment",
					Namespace:      "default",
					Notify: v1alpha1.NotifyConfig{
						Slack: "https://hooks.slack.com/test",
						Email: "test@example.com",
					},
				},
			}

			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec:   *tt.deploymentSpec,
				Status: *tt.deploymentStatus,
			}

			// 3. Fake 클라이언트 생성
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tracker, deployment).
				WithStatusSubresource(tracker).
				Build()

			// 4. Recorder 생성
			eventChan := make(chan string, 10)
			recorder := record.NewFakeRecorder(10)
			recorder.Events = eventChan

			// 5. Reconciler 생성
			r := &DeploymentTrackerReconciler{
				Client:   client,
				Scheme:   scheme,
				Recorder: recorder,
			}

			// 6. Reconcile 실행
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tracker.Name,
					Namespace: tracker.Namespace,
				},
			}

			// 7. CR이 정상적으로 생성되었는지 확인
			createdTracker := &v1alpha1.DeploymentTracker{}
			err := client.Get(ctx, req.NamespacedName, createdTracker)
			require.NoError(t, err, "Failed to get DeploymentTracker")

			result, err := r.Reconcile(ctx, req)

			// 8. 결과 확인
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// 상태 확인
				updatedTracker := &v1alpha1.DeploymentTracker{}
				err = client.Get(ctx, req.NamespacedName, updatedTracker)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedReady, updatedTracker.Status.Ready)

				// 이벤트 확인
				if tt.expectedReady {
					select {
					case event := <-eventChan:
						assert.Contains(t, event, "DeploymentReady")
					default:
						t.Error("Expected event not received")
					}
				}
			}
		})
	}
}

func TestReconcileEdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name         string
		setupFunc    func() (*v1alpha1.DeploymentTracker, *appsv1.Deployment)
		validateFunc func(*testing.T, *v1alpha1.DeploymentTracker, error)
	}{
		{
			name: "Deployment가 없는 경우",
			setupFunc: func() (*v1alpha1.DeploymentTracker, *appsv1.Deployment) {
				tracker := &v1alpha1.DeploymentTracker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tracker",
						Namespace: "default",
					},
					Spec: v1alpha1.DeploymentTrackerSpec{
						DeploymentName: "non-existent",
						Namespace:      "default",
					},
				}
				return tracker, nil
			},
			validateFunc: func(t *testing.T, tracker *v1alpha1.DeploymentTracker, err error) {
				assert.Error(t, err)
				assert.False(t, tracker.Status.Ready)
			},
		},
		{
			name: "Deployment에 Replicas가 설정되지 않은 경우",
			setupFunc: func() (*v1alpha1.DeploymentTracker, *appsv1.Deployment) {
				tracker := &v1alpha1.DeploymentTracker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-tracker",
						Namespace: "default",
					},
					Spec: v1alpha1.DeploymentTrackerSpec{
						DeploymentName: "test-deployment",
						Namespace:      "default",
					},
				}
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
				}
				return tracker, deployment
			},
			validateFunc: func(t *testing.T, tracker *v1alpha1.DeploymentTracker, err error) {
				assert.NoError(t, err)
				assert.False(t, tracker.Status.Ready)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tracker, deployment := tt.setupFunc()

			// Fake 클라이언트 설정
			clientBuilder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tracker)

			if deployment != nil {
				clientBuilder = clientBuilder.WithObjects(deployment)
			}

			client := clientBuilder.Build()

			// Reconciler 생성
			recorder := record.NewFakeRecorder(10)
			r := &DeploymentTrackerReconciler{
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

			_, err := r.Reconcile(ctx, req)

			// 결과 확인
			updatedTracker := &v1alpha1.DeploymentTracker{}
			_ = client.Get(ctx, req.NamespacedName, updatedTracker)
			tt.validateFunc(t, updatedTracker, err)
		})
	}
}
