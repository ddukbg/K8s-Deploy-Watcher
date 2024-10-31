// controllers/resource_tracker_controller.go

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types" // types import 추가
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ddukbgv1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

// ResourceTrackerReconciler reconciles a ResourceTracker object
type ResourceTrackerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ddukbgv1alpha1.ResourceTracker{}).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForResource),
		).
		Complete(r)
}

// findObjectsForResource maps a Deployment to its ResourceTracker
func (r *ResourceTrackerReconciler) findObjectsForResource(ctx context.Context, obj client.Object) []ctrl.Request {
	trackers := &ddukbgv1alpha1.ResourceTrackerList{}
	err := r.List(ctx, trackers)
	if err != nil {
		return nil
	}

	var requests []ctrl.Request
	for _, tracker := range trackers.Items {
		if tracker.Spec.Target.Kind == "Deployment" &&
			tracker.Spec.Target.Name == obj.GetName() &&
			tracker.Spec.Target.Namespace == obj.GetNamespace() {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tracker.Name,
					Namespace: tracker.Namespace,
				},
			})
		}
	}

	return requests
}

// Reconcile handles the main reconciliation logic
func (r *ResourceTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// ResourceTracker CR 가져오기
	tracker := &ddukbgv1alpha1.ResourceTracker{}
	if err := r.Get(ctx, req.NamespacedName, tracker); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 리소스 종류에 따른 처리
	switch tracker.Spec.Target.Kind {
	case "Deployment":
		return r.reconcileDeployment(ctx, tracker)
	case "StatefulSet":
		return ctrl.Result{}, fmt.Errorf("StatefulSet reconciliation not implemented yet")
	default:
		err := fmt.Errorf("unsupported resource kind: %s", tracker.Spec.Target.Kind)
		logger.Error(err, "Unsupported resource kind")
		return ctrl.Result{}, err
	}
}

// sendSlackNotification sends a notification to Slack
func sendSlackNotification(webhookURL, message string) error {
	payload := map[string]string{
		"text": message,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

// reconcileDeployment handles Deployment type resources
func (r *ResourceTrackerReconciler) reconcileDeployment(ctx context.Context, tracker *ddukbgv1alpha1.ResourceTracker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Deployment 가져오기
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, deployment); err != nil {
		logger.Error(err, "Failed to fetch Deployment")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// 현재 이미지 가져오기
	currentImage := ""
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		currentImage = deployment.Spec.Template.Spec.Containers[0].Image
	}

	// 이전 이미지와 비교
	previousImage, ok := deployment.Annotations["ddukbg.k8s/previous-image"]
	if !ok {
		// 최초 실행 시
		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}
		deployment.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(ctx, deployment); err != nil {
			logger.Error(err, "Failed to update deployment annotations")
			return ctrl.Result{}, err
		}
	} else if previousImage != currentImage {
		// 이미지가 변경됨
		tracker.Status.LastUpdated = &metav1.Time{Time: time.Now()}
		tracker.Status.Message = fmt.Sprintf("Image updated from %s to %s", previousImage, currentImage)

		// Deployment 주석 업데이트
		deployment.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(ctx, deployment); err != nil {
			logger.Error(err, "Failed to update deployment annotations")
			return ctrl.Result{}, err
		}

		// Status 업데이트
		if err := r.Status().Update(ctx, tracker); err != nil {
			logger.Error(err, "Failed to update ResourceTracker status")
			return ctrl.Result{}, err
		}

		// Slack 알림 전송
		if tracker.Spec.Notify.Slack != "" {
			message := fmt.Sprintf("*Deployment %s/%s image updated*\n"+
				"> Previous Image: %s\n"+
				"> New Image: %s",
				deployment.Namespace, deployment.Name,
				previousImage, currentImage)

			if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
				logger.Error(err, "Failed to send Slack notification")
			}
		}
	}

	// Deployment 상태 확인
	isReady := deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
		deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
		deployment.Status.AvailableReplicas == *deployment.Spec.Replicas

	if tracker.Status.Ready != isReady {
		tracker.Status.Ready = isReady
		tracker.Status.CurrentState.ReadyReplicas = deployment.Status.ReadyReplicas
		tracker.Status.CurrentState.TotalReplicas = *deployment.Spec.Replicas

		if isReady {
			tracker.Status.Message = "Resource is running successfully"
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "ResourceReady",
				fmt.Sprintf("Resource %s is running successfully", deployment.Name))
		} else {
			tracker.Status.Message = "Resource is not ready"
		}

		if err := r.Status().Update(ctx, tracker); err != nil {
			logger.Error(err, "Failed to update ResourceTracker status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// detectImageChange checks if the deployment's image has changed
func (r *ResourceTrackerReconciler) detectImageChange(deployment *appsv1.Deployment) (bool, string, string) {
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return false, "", ""
	}

	currentImage := deployment.Spec.Template.Spec.Containers[0].Image
	annotations := deployment.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
		deployment.Annotations = annotations
	}

	previousImage := annotations["ddukbg.k8s/previous-image"]
	if previousImage == "" {
		// 최초 실행시
		deployment.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(context.Background(), deployment); err != nil {
			return false, "", ""
		}
		return false, currentImage, currentImage
	}

	if previousImage != currentImage {
		deployment.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(context.Background(), deployment); err != nil {
			return false, "", ""
		}
		return true, previousImage, currentImage
	}

	return false, currentImage, currentImage
}
