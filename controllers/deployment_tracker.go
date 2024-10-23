package controllers

import (
	"context"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

type DeploymentTrackerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentTrackerReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Custom Resource 가져오기
	tracker := &v1alpha1.DeploymentTracker{}
	err := r.Get(ctx, req.NamespacedName, tracker)
	if err != nil {
		logger.Error(err, "Failed to fetch DeploymentTracker CR")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// 2. 해당 Deployment의 상태 확인
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: tracker.Spec.DeploymentName, Namespace: tracker.Spec.Namespace}, deployment)
	if err != nil {
		logger.Error(err, "Failed to fetch Deployment")
		return reconcile.Result{}, err
	}

	// 3. 배포가 성공적으로 완료되었는지 확인
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		logger.Info("Deployment is running successfully", "Deployment", deployment.Name)

		// 4. 알림 전송
		if tracker.Spec.Notify.Slack != "" {
			err := sendSlackNotification(tracker.Spec.Notify.Slack, fmt.Sprintf("Deployment %s is successfully running", deployment.Name))
			if err != nil {
				logger.Error(err, "Failed to send Slack notification")
			}
		}
	}

	return reconcile.Result{}, nil
}

func sendSlackNotification(webhookURL, message string) error {
	payload := map[string]string{"text": message}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	return nil
}
