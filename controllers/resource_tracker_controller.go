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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// SlackMessage 구조체 정의 추가
type SlackMessage struct {
	Text string `json:"text"`
}

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
		Watches(
			&appsv1.StatefulSet{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForResource),
		).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForResource),
		).
		Complete(r)
}

// findObjectsForResource finds ResourceTrackers that monitor the given resource
func (r *ResourceTrackerReconciler) findObjectsForResource(ctx context.Context, obj client.Object) []ctrl.Request {
	trackers := &ddukbgv1alpha1.ResourceTrackerList{}
	err := r.List(ctx, trackers)
	if err != nil {
		return nil
	}

	var requests []ctrl.Request
	for _, tracker := range trackers.Items {
		// 리소스 종류 확인
		if (tracker.Spec.Target.Kind == "Deployment" && obj.GetObjectKind().GroupVersionKind().Kind == "Deployment") ||
			(tracker.Spec.Target.Kind == "StatefulSet" && obj.GetObjectKind().GroupVersionKind().Kind == "StatefulSet") ||
			(tracker.Spec.Target.Kind == "Pod" && obj.GetObjectKind().GroupVersionKind().Kind == "Pod") {

			// 네임스페이스 확인
			if tracker.Spec.Target.Namespace == obj.GetNamespace() {
				// 특정 리소스 이름이 지정되었다면 이름도 확인
				if tracker.Spec.Target.Name == "" || tracker.Spec.Target.Name == obj.GetName() {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      tracker.Name,
							Namespace: tracker.Namespace,
						},
					})
				}
			}
		}
	}
	return requests
}

// Reconcile 함수 수정
func (r *ResourceTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	tracker := &ddukbgv1alpha1.ResourceTracker{}

	if err := r.Get(ctx, req.NamespacedName, tracker); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 지원하지 않는 리소스 종류 체크
	if tracker.Spec.Target.Kind != "Deployment" && tracker.Spec.Target.Kind != "StatefulSet" {
		return ctrl.Result{}, fmt.Errorf("unsupported resource kind: %s", tracker.Spec.Target.Kind)
	}

	// 리소스 상태 체크
	isReady := false
	statusChanged := false

	switch tracker.Spec.Target.Kind {
	case "Deployment":
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: tracker.Spec.Target.Namespace,
			Name:      tracker.Spec.Target.Name,
		}, deploy); err != nil {
			return ctrl.Result{}, err
		}

		isReady = deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
			deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
			deploy.Status.AvailableReplicas == *deploy.Spec.Replicas

		key := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)
		if tracker.Status.ResourceStatus == nil {
			tracker.Status.ResourceStatus = make(map[string]bool)
		}

		if tracker.Status.ResourceStatus[key] != isReady {
			statusChanged = true
			tracker.Status.ResourceStatus[key] = isReady

			if isReady {
				msg := fmt.Sprintf("Deployment %s is ready", key)
				r.Recorder.Event(tracker, "Normal", "ResourceReady", msg)

				if tracker.Spec.Notify.Slack != "" {
					slackMsg := formatSlackMessage("Deployment", deploy.Namespace, deploy.Name,
						deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
					if err := sendSlackNotification(tracker.Spec.Notify.Slack, slackMsg); err != nil {
						logger.Error(err, "Failed to send Slack notification")
					}
				}
			}
		}

	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: tracker.Spec.Target.Namespace,
			Name:      tracker.Spec.Target.Name,
		}, sts); err != nil {
			return ctrl.Result{}, err
		}

		isReady = sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
			sts.Status.UpdatedReplicas == *sts.Spec.Replicas

		key := fmt.Sprintf("%s/%s", sts.Namespace, sts.Name)
		if tracker.Status.ResourceStatus == nil {
			tracker.Status.ResourceStatus = make(map[string]bool)
		}

		if tracker.Status.ResourceStatus[key] != isReady {
			statusChanged = true
			tracker.Status.ResourceStatus[key] = isReady

			if isReady {
				msg := fmt.Sprintf("StatefulSet %s is ready", key)
				r.Recorder.Event(tracker, "Normal", "ResourceReady", msg)

				if tracker.Spec.Notify.Slack != "" {
					slackMsg := formatSlackMessage("StatefulSet", sts.Namespace, sts.Name,
						sts.Status.ReadyReplicas, *sts.Spec.Replicas)
					if err := sendSlackNotification(tracker.Spec.Notify.Slack, slackMsg); err != nil {
						logger.Error(err, "Failed to send Slack notification")
					}
				}
			}
		}
	}

	// 상태 업데이트
	if statusChanged {
		if err := r.Status().Update(ctx, tracker); err != nil {
			logger.Error(err, "Failed to update ResourceTracker status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// sendSlackNotification sends a notification to Slack
var sendSlackNotification = func(webhookURL string, message string) error {
	payload := SlackMessage{
		Text: message,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status code: %d", resp.StatusCode)
	}

	return nil
}

// reconcileDeployment handles Deployment type resources
func (r *ResourceTrackerReconciler) reconcileDeployment(ctx context.Context, tracker *ddukbgv1alpha1.ResourceTracker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if tracker.Spec.Target.Name == "" {
		deployList := &appsv1.DeploymentList{}
		if err := r.List(ctx, deployList, client.InNamespace(tracker.Spec.Target.Namespace)); err != nil {
			logger.Error(err, "Failed to list Deployments")
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}

		statusChanged := false
		readyDeployments := make(map[string]bool)

		for _, deploy := range deployList.Items {
			isReady := deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
				deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas

			key := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)

			// Generation 변경 확인
			currentGeneration := fmt.Sprintf("%d-%d", deploy.Generation, deploy.Status.ObservedGeneration)
			prevGeneration, hasPrevGen := tracker.Status.GenerationStatus[key]

			// 상태가 변경되었거나 Generation이 변경된 경우에만 알림
			if isReady {
				shouldNotify := false

				// 이전에 없던 새로운 리소스
				if !hasPrevGen {
					shouldNotify = true
				} else if prevGeneration != currentGeneration {
					// Generation이 변경된 경우
					shouldNotify = true
				}

				if shouldNotify {
					if tracker.Spec.Notify.Slack != "" {
						message := fmt.Sprintf("Deployment %s/%s is now ready\n"+
							"> Namespace: %s\n"+
							"> Status: Running\n"+
							"> Replicas: %d/%d ready",
							deploy.Namespace, deploy.Name,
							deploy.Namespace,
							deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
						if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
							logger.Error(err, "Failed to send Slack notification")
						}
					}

					r.Recorder.Event(tracker, corev1.EventTypeNormal, "ResourceReady",
						fmt.Sprintf("Deployment %s is running successfully", deploy.Name))
				}
			}

			// 상태 업데이트
			readyDeployments[key] = isReady
			if tracker.Status.GenerationStatus == nil {
				tracker.Status.GenerationStatus = make(map[string]string)
			}
			if tracker.Status.GenerationStatus[key] != currentGeneration {
				statusChanged = true
				tracker.Status.GenerationStatus[key] = currentGeneration
			}
		}

		// 상태가 변경된 경우에만 상태 업데이트
		if statusChanged {
			if tracker.Status.ResourceStatus == nil {
				tracker.Status.ResourceStatus = make(map[string]bool)
			}
			for k, v := range readyDeployments {
				tracker.Status.ResourceStatus[k] = v
			}
			if err := r.Status().Update(ctx, tracker); err != nil {
				logger.Error(err, "Failed to update ResourceTracker status")
				return ctrl.Result{RequeueAfter: time.Second * 30}, nil
			}
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *ResourceTrackerReconciler) reconcileStatefulSet(ctx context.Context, tracker *ddukbgv1alpha1.ResourceTracker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 네임스페이스 전체 모니터링인 경우
	if tracker.Spec.Target.Name == "" {
		stsList := &appsv1.StatefulSetList{}
		if err := r.List(ctx, stsList, client.InNamespace(tracker.Spec.Target.Namespace)); err != nil {
			logger.Error(err, "Failed to list StatefulSets")
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}

		// 각 StatefulSet 개별 처리
		for _, sts := range stsList.Items {
			isReady := sts.Status.ReadyReplicas == *sts.Spec.Replicas

			if isReady {
				// 개별 StatefulSet에 대한 알림
				if tracker.Spec.Notify.Slack != "" {
					message := fmt.Sprintf("StatefulSet %s/%s is now ready\n"+
						"> Namespace: %s\n"+
						"> Status: Running\n"+
						"> Replicas: %d/%d ready",
						sts.Namespace, sts.Name,
						sts.Namespace,
						sts.Status.ReadyReplicas, *sts.Spec.Replicas)
					if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
						logger.Error(err, "Failed to send Slack notification")
					}
				}

				// 개별 이벤트 발생
				r.Recorder.Event(tracker, corev1.EventTypeNormal, "ResourceReady",
					fmt.Sprintf("StatefulSet %s is running successfully", sts.Name))
			}
		}
	}

	// 단일 StatefulSet 처리 로직
	// StatefulSet이 없는 경우도 정상 케이스로 처리
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, sts); err != nil {
		if apierrors.IsNotFound(err) {
			// StatefulSet이 없는 경우 정상 처리
			tracker.Status.Message = "No StatefulSet found"
			tracker.Status.Ready = true
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.Status().Update(ctx, tracker)
		}
		logger.Error(err, "Failed to fetch StatefulSet")
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	// 현재 이미지 가져오
	currentImage := ""
	if len(sts.Spec.Template.Spec.Containers) > 0 {
		currentImage = sts.Spec.Template.Spec.Containers[0].Image
	}

	// 이전 이미지와 비교
	previousImage, ok := sts.Annotations["ddukbg.k8s/previous-image"]
	if !ok {
		// 최초 실행 시
		if sts.Annotations == nil {
			sts.Annotations = make(map[string]string)
		}
		sts.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(ctx, sts); err != nil {
			logger.Error(err, "Failed to update statefulset annotations")
			return ctrl.Result{}, err
		}
	} else if previousImage != currentImage {
		// 이미지가 변경됨
		tracker.Status.LastUpdated = &metav1.Time{Time: time.Now()}
		tracker.Status.Message = fmt.Sprintf("Image updated from %s to %s", previousImage, currentImage)

		// StatefulSet 주석 업데이트
		sts.Annotations["ddukbg.k8s/previous-image"] = currentImage
		if err := r.Update(ctx, sts); err != nil {
			logger.Error(err, "Failed to update statefulset annotations")
			return ctrl.Result{}, err
		}

		// Status 업데이트
		if err := r.Status().Update(ctx, tracker); err != nil {
			logger.Error(err, "Failed to update ResourceTracker status")
			return ctrl.Result{}, err
		}

		// Slack 알림 전송
		if tracker.Spec.Notify.Slack != "" {
			message := fmt.Sprintf("*StatefulSet %s/%s image updated*\n"+
				"> Previous Image: %s\n"+
				"> New Image: %s",
				sts.Namespace, sts.Name,
				previousImage, currentImage)

			if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
				logger.Error(err, "Failed to send Slack notification")
			}
		}
	}

	// StatefulSet 상태 확인
	isReady := sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
		sts.Status.CurrentReplicas == *sts.Spec.Replicas &&
		sts.Status.UpdatedReplicas == *sts.Spec.Replicas

	if tracker.Status.Ready != isReady {
		tracker.Status.Ready = isReady
		tracker.Status.CurrentState.ReadyReplicas = sts.Status.ReadyReplicas
		tracker.Status.CurrentState.TotalReplicas = *sts.Spec.Replicas

		if isReady {
			tracker.Status.Message = "Resource is running successfully"
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "ResourceReady",
				fmt.Sprintf("Resource %s is running successfully", sts.Name))

			if tracker.Spec.Notify.Slack != "" {
				message := fmt.Sprintf("*StatefulSet %s/%s is now ready*\n"+
					"> Status: Running\n"+
					"> Ready Replicas: %d/%d",
					sts.Namespace, sts.Name,
					sts.Status.ReadyReplicas,
					*sts.Spec.Replicas)

				if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
					logger.Error(err, "Failed to send Slack notification")
				}
			}
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

// reconcilePod handles Pod type resources
func (r *ResourceTrackerReconciler) reconcilePod(ctx context.Context, tracker *ddukbgv1alpha1.ResourceTracker) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Pod가 없는 경우도 정상 케이스로 처리
	pod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, pod); err != nil {
		if apierrors.IsNotFound(err) {
			tracker.Status.Message = "No Pod found"
			tracker.Status.Ready = true
			return ctrl.Result{RequeueAfter: time.Second * 30}, r.Status().Update(ctx, tracker)
		}
		logger.Error(err, "Failed to fetch Pod")
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	// Pod 상태 확인
	isReady := pod.Status.Phase == corev1.PodRunning
	if tracker.Status.Ready != isReady {
		tracker.Status.Ready = isReady
		tracker.Status.CurrentState.ReadyReplicas = 1
		tracker.Status.CurrentState.TotalReplicas = 1

		if isReady {
			tracker.Status.Message = "Pod is running successfully"
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "PodReady",
				fmt.Sprintf("Pod %s is running successfully", pod.Name))

			// Slack 알림 전송
			if tracker.Spec.Notify.Slack != "" {
				message := fmt.Sprintf("*Pod %s/%s is now ready*\n"+
					"> Status: Running\n"+
					"> Phase: %s",
					pod.Namespace, pod.Name,
					pod.Status.Phase)

				if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
					logger.Error(err, "Failed to send Slack notification")
				}
			}
		} else {
			tracker.Status.Message = fmt.Sprintf("Pod is not ready: %s", pod.Status.Phase)
		}

		if err := r.Status().Update(ctx, tracker); err != nil {
			logger.Error(err, "Failed to update ResourceTracker status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *ResourceTrackerReconciler) checkDeploymentStatus(ctx context.Context, tracker *ddukbgv1alpha1.ResourceTracker, deploy *appsv1.Deployment) (bool, bool) {
	isReady := deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
		deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
		deploy.Status.AvailableReplicas == *deploy.Spec.Replicas

	key := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)
	statusChanged := tracker.Status.ResourceStatus[key] != isReady

	if statusChanged {
		if tracker.Status.ResourceStatus == nil {
			tracker.Status.ResourceStatus = make(map[string]bool)
		}
		tracker.Status.ResourceStatus[key] = isReady

		if isReady {
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "ResourceReady",
				fmt.Sprintf("Deployment %s is running successfully", deploy.Name))

			if tracker.Spec.Notify.Slack != "" {
				message := fmt.Sprintf("*Deployment %s/%s is now ready*\n"+
					"> Namespace: %s\n"+
					"> Status: Running\n"+
					"> Replicas: %d/%d ready",
					deploy.Namespace, deploy.Name,
					deploy.Namespace,
					deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
				if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
					return isReady, false
				}
			}
		}
	}

	return isReady, statusChanged
}

// Slack 메시지 포맷 함수
func formatSlackMessage(kind, namespace, name string, readyReplicas, totalReplicas int32) string {
	return fmt.Sprintf("%s %s/%s is now ready\n"+
		"> Namespace: %s\n"+
		"> Status: Running\n"+
		"> Replicas: %d/%d ready",
		kind, namespace, name,
		namespace,
		readyReplicas, totalReplicas)
}
