package controllers

import (
    "context"
    "testing"

    appsv1 "k8s.io/api/apps/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "github.com/stretchr/testify/assert"
)

func TestReconcile_Success(t *testing.T) {
    // 1. 가짜 클라이언트 생성 (fake.Client)
    scheme := runtime.NewScheme()
    _ = appsv1.AddToScheme(scheme)  // 필요한 스키마 추가

    // 테스트용 Deployment 생성
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "my-app",
            Namespace: "default",
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(2), // 2개의 파드
        },
        Status: appsv1.DeploymentStatus{
            ReadyReplicas: 2,
        },
    }

    // 가짜 클라이언트에 Deployment 추가
    client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment).Build()

    // 2. Reconciler 생성
    r := &DeploymentTrackerReconciler{
        Client: client,
        Scheme: scheme,
    }

    // 3. Reconcile 호출
    result, err := r.Reconcile(context.TODO(), reconcile.Request{
        NamespacedName: types.NamespacedName{
            Name:      "my-app",
            Namespace: "default",
        },
    })

    // 4. 테스트 결과 확인
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.False(t, result.Requeue) // Requeue가 False여야 함 (정상 배포)
}

// 헬퍼 함수
func int32Ptr(i int32) *int32 { return &i }
