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

	// 상태 맵 초기화
	if tracker.Status.ResourceStatus == nil {
		tracker.Status.ResourceStatus = make(map[string]bool)
	}

	var result ctrl.Result
	var err error

	switch tracker.Spec.Target.Kind {
	case "Deployment":
		result, err = r.reconcileDeployment(ctx, tracker)
	case "StatefulSet":
		result, err = r.reconcileStatefulSet(ctx, tracker)
	case "Pod":
		result, err = r.reconcilePod(ctx, tracker)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported resource kind: %s", tracker.Spec.Target.Kind)
	}

	if err != nil {
		logger.Error(err, "Failed to reconcile resource")
		return ctrl.Result{}, err
	}

	return result, nil
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

	// 네임스페이스 전체 모니터링인 경우
	if tracker.Spec.Target.Name == "" {
		deployList := &appsv1.DeploymentList{}
		if err := r.List(ctx, deployList, client.InNamespace(tracker.Spec.Target.Namespace)); err != nil {
			return ctrl.Result{}, err
		}

		statusChanged := false
		readyDeployments := 0
		totalDeployments := len(deployList.Items)

		for _, deploy := range deployList.Items {
			key := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)
			isReady := deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
				deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
				deploy.Status.AvailableReplicas == *deploy.Spec.Replicas

			if isReady {
				readyDeployments++
			}

			if tracker.Status.ResourceStatus[key] != isReady {
				statusChanged = true
				tracker.Status.ResourceStatus[key] = isReady

				if isReady {
					r.Recorder.Event(tracker, corev1.EventTypeNormal, "DeploymentReady",
						fmt.Sprintf("Deployment %s is ready", key))

					if tracker.Spec.Notify.Slack != "" {
						message := formatSlackMessage("Deployment", deploy.Namespace, deploy.Name,
							deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
						if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
							logger.Error(err, "Failed to send Slack notification")
						}
					}
				}
			}
		}

		if statusChanged {
			tracker.Status.CurrentState.ReadyReplicas = int32(readyDeployments)
			tracker.Status.CurrentState.TotalReplicas = int32(totalDeployments)
			if err := r.Status().Update(ctx, tracker); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// 단일 Deployment 모니터링 로직
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, deploy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isReady := deploy.Status.ReadyReplicas == *deploy.Spec.Replicas &&
		deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
		deploy.Status.AvailableReplicas == *deploy.Spec.Replicas

	key := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)
	if tracker.Status.ResourceStatus[key] != isReady {
		tracker.Status.ResourceStatus[key] = isReady
		tracker.Status.CurrentState.ReadyReplicas = deploy.Status.ReadyReplicas
		tracker.Status.CurrentState.TotalReplicas = *deploy.Spec.Replicas

		if isReady {
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "DeploymentReady",
				fmt.Sprintf("Deployment %s is ready", key))

			if tracker.Spec.Notify.Slack != "" {
				message := formatSlackMessage("Deployment", deploy.Namespace, deploy.Name,
					deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
				if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
					logger.Error(err, "Failed to send Slack notification")
				}
			}
		}

		if err := r.Status().Update(ctx, tracker); err != nil {
			return ctrl.Result{}, err
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
			return ctrl.Result{}, err
		}

		statusChanged := false
		readySts := 0
		totalSts := len(stsList.Items)

		for _, sts := range stsList.Items {
			key := fmt.Sprintf("%s/%s", sts.Namespace, sts.Name)
			isReady := sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
				sts.Status.UpdatedReplicas == *sts.Spec.Replicas

			if isReady {
				readySts++
			}

			if tracker.Status.ResourceStatus[key] != isReady {
				statusChanged = true
				tracker.Status.ResourceStatus[key] = isReady

				if isReady {
					r.Recorder.Event(tracker, corev1.EventTypeNormal, "StatefulSetReady",
						fmt.Sprintf("StatefulSet %s is ready", key))

					if tracker.Spec.Notify.Slack != "" {
						message := formatSlackMessage("StatefulSet", sts.Namespace, sts.Name,
							sts.Status.ReadyReplicas, *sts.Spec.Replicas)
						if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
							logger.Error(err, "Failed to send Slack notification")
						}
					}
				}
			}
		}

		if statusChanged {
			tracker.Status.CurrentState.ReadyReplicas = int32(readySts)
			tracker.Status.CurrentState.TotalReplicas = int32(totalSts)
			if err := r.Status().Update(ctx, tracker); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// 단일 StatefulSet 모니터링 로직
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isReady := sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
		sts.Status.UpdatedReplicas == *sts.Spec.Replicas

	key := fmt.Sprintf("%s/%s", sts.Namespace, sts.Name)
	if tracker.Status.ResourceStatus[key] != isReady {
		tracker.Status.ResourceStatus[key] = isReady
		tracker.Status.CurrentState.ReadyReplicas = sts.Status.ReadyReplicas
		tracker.Status.CurrentState.TotalReplicas = *sts.Spec.Replicas

		if isReady {
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "StatefulSetReady",
				fmt.Sprintf("StatefulSet %s is ready", key))

			if tracker.Spec.Notify.Slack != "" {
				message := formatSlackMessage("StatefulSet", sts.Namespace, sts.Name,
					sts.Status.ReadyReplicas, *sts.Spec.Replicas)
				if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
					logger.Error(err, "Failed to send Slack notification")
				}
			}
		}

		if err := r.Status().Update(ctx, tracker); err != nil {
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

	// 상태 맵 초기화
	if tracker.Status.ResourceStatus == nil {
		tracker.Status.ResourceStatus = make(map[string]bool)
	}

	// 네임스페이스 전체 모니터링인 경우
	if tracker.Spec.Target.Name == "" {
		podList := &corev1.PodList{}
		if err := r.List(ctx, podList, client.InNamespace(tracker.Spec.Target.Namespace)); err != nil {
			logger.Error(err, "Failed to list Pods")
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}

		statusChanged := false
		readyPods := 0
		totalPods := len(podList.Items)

		// 각 Pod 개별 처리
		for _, pod := range podList.Items {
			key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
			isReady := pod.Status.Phase == corev1.PodRunning

			if isReady {
				readyPods++
			}

			if tracker.Status.ResourceStatus[key] != isReady {
				statusChanged = true
				tracker.Status.ResourceStatus[key] = isReady

				if isReady {
					r.Recorder.Event(tracker, corev1.EventTypeNormal, "PodReady",
						fmt.Sprintf("Pod %s is running successfully", pod.Name))

					if tracker.Spec.Notify.Slack != "" {
						message := fmt.Sprintf("Pod %s/%s is now ready\n"+
							"> Namespace: %s\n"+
							"> Status: Running\n"+
							"> Phase: %s",
							pod.Namespace, pod.Name,
							pod.Namespace,
							pod.Status.Phase)
						if err := sendSlackNotification(tracker.Spec.Notify.Slack, message); err != nil {
							logger.Error(err, "Failed to send Slack notification")
						}
					}
				}
			}
		}

		// 전체 상태 업데이트
		if statusChanged {
			tracker.Status.CurrentState.ReadyReplicas = int32(readyPods)
			tracker.Status.CurrentState.TotalReplicas = int32(totalPods)
			tracker.Status.Message = fmt.Sprintf("%d/%d pods are running", readyPods, totalPods)

			if err := r.Status().Update(ctx, tracker); err != nil {
				logger.Error(err, "Failed to update ResourceTracker status")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// 단일 Pod 모니터링인 경우
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      tracker.Spec.Target.Name,
		Namespace: tracker.Spec.Target.Namespace,
	}, pod); err != nil {
		if apierrors.IsNotFound(err) {
			tracker.Status.Message = "Pod not found"
			if err := r.Status().Update(ctx, tracker); err != nil {
				logger.Error(err, "Failed to update ResourceTracker status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}
		logger.Error(err, "Failed to get Pod")
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	// Pod 상태 확인
	key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	isReady := pod.Status.Phase == corev1.PodRunning
	statusChanged := tracker.Status.ResourceStatus[key] != isReady

	if statusChanged {
		tracker.Status.ResourceStatus[key] = isReady
		tracker.Status.CurrentState.ReadyReplicas = boolToInt32(isReady)
		tracker.Status.CurrentState.TotalReplicas = 1

		if isReady {
			tracker.Status.Message = "Pod is running successfully"
			r.Recorder.Event(tracker, corev1.EventTypeNormal, "PodReady",
				fmt.Sprintf("Pod %s is running successfully", pod.Name))

			if tracker.Spec.Notify.Slack != "" {
				message := fmt.Sprintf("Pod %s/%s is now ready\n"+
					"> Namespace: %s\n"+
					"> Status: Running\n"+
					"> Phase: %s",
					pod.Namespace, pod.Name,
					pod.Namespace,
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

// bool을 int32로 변환하는 헬퍼 함수
func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
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
