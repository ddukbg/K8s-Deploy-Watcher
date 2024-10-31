// controllers/resource_tracker_controller_test.go

package controllers

import (
	"context"
	"testing"
	"time"

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

	ddukbgv1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

func TestReconcileResourceTracker(t *testing.T) {
	// 테스트 케이스 정의
	tests := []struct {
		name          string
		kind          string
		initialImage  string
		updatedImage  string
		replicas      int32
		readyReplicas int32
		expectError   bool
		expectReady   bool
	}{
		{
			name:          "정상적인 Deployment 추적",
			kind:          "Deployment",
			initialImage:  "nginx:1.19",
			updatedImage:  "nginx:1.20",
			replicas:      3,
			readyReplicas: 3,
			expectError:   false,
			expectReady:   true,
		},
		{
			name:          "배포 진행 중",
			kind:          "Deployment",
			initialImage:  "nginx:1.19",
			updatedImage:  "nginx:1.19",
			replicas:      3,
			readyReplicas: 1,
			expectError:   false,
			expectReady:   false,
		},
		{
			name:          "지원하지 않는 리소스 종류",
			kind:          "DaemonSet",
			initialImage:  "nginx:1.19",
			updatedImage:  "nginx:1.19",
			replicas:      1,
			readyReplicas: 1,
			expectError:   true,
			expectReady:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// 1. 테스트 환경 설정
			scheme := runtime.NewScheme()
			_ = ddukbgv1alpha1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			// 2. ResourceTracker CR 생성
			tracker := &ddukbgv1alpha1.ResourceTracker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tracker",
					Namespace: "default",
				},
				Spec: ddukbgv1alpha1.ResourceTrackerSpec{
					Target: ddukbgv1alpha1.ResourceTarget{
						Kind:      tt.kind,
						Name:      "test-app",
						Namespace: "default",
					},
					Notify: ddukbgv1alpha1.NotifyConfig{
						Slack: "https://hooks.slack.com/test",
					},
				},
			}

			// 3. Deployment 생성
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &tt.replicas,
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
									Image: tt.initialImage,
								},
							},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          tt.replicas,
					ReadyReplicas:     tt.readyReplicas,
					UpdatedReplicas:   tt.replicas,
					AvailableReplicas: tt.readyReplicas,
				},
			}

			// 4. Fake 클라이언트 생성
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tracker, deployment).
				WithStatusSubresource(tracker, deployment).
				Build()

			// 5. Recorder 설정
			recorder := record.NewFakeRecorder(10)

			// 6. Reconciler 생성
			r := &ResourceTrackerReconciler{
				Client:   client,
				Scheme:   scheme,
				Recorder: recorder,
			}

			// 7. Reconcile 실행
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tracker.Name,
					Namespace: tracker.Namespace,
				},
			}

			result, err := r.Reconcile(ctx, req)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// 상태 확인
			updatedTracker := &ddukbgv1alpha1.ResourceTracker{}
			err = client.Get(ctx, req.NamespacedName, updatedTracker)
			require.NoError(t, err)
			assert.Equal(t, tt.expectReady, updatedTracker.Status.Ready)

			// 이미지 변경 테스트
			if tt.initialImage != tt.updatedImage {
				// Get latest deployment
				updatedDeployment := &appsv1.Deployment{}
				err = client.Get(ctx, types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				}, updatedDeployment)
				require.NoError(t, err)

				// Update image
				updatedDeployment.Spec.Template.Spec.Containers[0].Image = tt.updatedImage
				err = client.Update(ctx, updatedDeployment)
				require.NoError(t, err)

				// Wait a bit to simulate real-world scenario
				time.Sleep(time.Millisecond * 100)

				// Reconcile again
				result, err = r.Reconcile(ctx, req)
				assert.NoError(t, err)

				// Check tracker status
				finalTracker := &ddukbgv1alpha1.ResourceTracker{}
				err = client.Get(ctx, req.NamespacedName, finalTracker)
				require.NoError(t, err)
				assert.Contains(t, finalTracker.Status.Message, "Image updated")
			}
		})
	}
}

func TestReconcileEvents(t *testing.T) {
	// 1. 테스트 환경 설정
	scheme := runtime.NewScheme()
	_ = ddukbgv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// 2. ResourceTracker 생성
	tracker := &ddukbgv1alpha1.ResourceTracker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tracker",
			Namespace: "default",
		},
		Spec: ddukbgv1alpha1.ResourceTrackerSpec{
			Target: ddukbgv1alpha1.ResourceTarget{
				Kind:      "Deployment",
				Name:      "test-app",
				Namespace: "default",
			},
			Notify: ddukbgv1alpha1.NotifyConfig{
				Slack: "https://hooks.slack.com/test",
			},
		},
	}

	// 3. Deployment 생성
	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
							Name:  "test-container",
							Image: "nginx:1.19",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          replicas,
			ReadyReplicas:     replicas,
			UpdatedReplicas:   replicas,
			AvailableReplicas: replicas,
		},
	}

	// 4. 테스트 실행
	recorder := record.NewFakeRecorder(10)
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tracker, deployment).
		WithStatusSubresource(tracker, deployment).
		Build()

	r := &ResourceTrackerReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
	}

	// 5. Reconcile 실행
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      tracker.Name,
			Namespace: tracker.Namespace,
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// 6. 이벤트 확인
	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, "ResourceReady")
	case <-time.After(time.Second * 5):
		t.Error("Expected event not received")
	}
}
