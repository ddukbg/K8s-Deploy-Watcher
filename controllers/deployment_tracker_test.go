package controllers

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    appsv1 "k8s.io/api/apps/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"

    v1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

func TestReconcile(t *testing.T) {
    // 테스트 케이스들
    tests := []struct {
        name           string
        deploymentSpec *appsv1.DeploymentSpec
        deploymentStatus *appsv1.DeploymentStatus
        expectedReady  bool
        expectError    bool
    }{
        {
            name: "성공적인 배포",
            deploymentSpec: &appsv1.DeploymentSpec{
                Replicas: int32Ptr(3),
            },
            deploymentStatus: &appsv1.DeploymentStatus{
                ReadyReplicas: 3,
            },
            expectedReady: true,
            expectError: false,
        },
        {
            name: "진행 중인 배포",
            deploymentSpec: &appsv1.DeploymentSpec{
                Replicas: int32Ptr(3),
            },
            deploymentStatus: &appsv1.DeploymentStatus{
                ReadyReplicas: 1,
            },
            expectedReady: false,
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 1. 테스트 환경 설정
            scheme := runtime.NewScheme()
            _ = appsv1.AddToScheme(scheme)
            _ = v1alpha1.AddToScheme(scheme)

            // 2. 테스트 객체 생성
            tracker := &v1alpha1.DeploymentTracker{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-tracker",
                    Namespace: "default",
                },
                Spec: v1alpha1.DeploymentTrackerSpec{
                    DeploymentName: "test-deployment",
                    Namespace:     "default",
                    Notify: v1alpha1.Notify{
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
                Build()

            // 4. Reconciler 생성
            r := &DeploymentTrackerReconciler{
                Client: client,
                Scheme: scheme,
            }

            // 5. Reconcile 실행
            req := reconcile.Request{
                NamespacedName: types.NamespacedName{
                    Name:      tracker.Name,
                    Namespace: tracker.Namespace,
                },
            }
            result, err := r.Reconcile(context.Background(), req)

            // 6. 결과 확인
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)

                // 상태 확인
                updatedTracker := &v1alpha1.DeploymentTracker{}
                err = client.Get(context.Background(), req.NamespacedName, updatedTracker)
                require.NoError(t, err)
                assert.Equal(t, tt.expectedReady, updatedTracker.Status.Ready)
            }
        })
    }
}

func int32Ptr(i int32) *int32 { return &i }