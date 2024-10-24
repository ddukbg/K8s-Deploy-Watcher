package controllers

import (
    "context"
    "fmt"
    "net/http"
    "bytes"
    "encoding/json"
    "net/smtp"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/client-go/tools/record"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/source"
    "sigs.k8s.io/controller-runtime/pkg/handler"

    v1alpha1 "k8s-deploy-watcher/api/v1alpha1"
)

type DeploymentTrackerReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.DeploymentTracker{}).
        Watches(
            &source.Kind{Type: &appsv1.Deployment{}},
            handler.EnqueueRequestsFromMapFunc(r.findObjectsForDeployment),
        ).
        Complete(r)
}

func (r *DeploymentTrackerReconciler) findObjectsForDeployment(ctx context.Context, obj client.Object) []reconcile.Request {
    trackers := &v1alpha1.DeploymentTrackerList{}
    err := r.List(ctx, trackers)
    if err != nil {
        return nil
    }

    requests := make([]reconcile.Request, 0)
    for _, tracker := range trackers.Items {
        if tracker.Spec.DeploymentName == obj.GetName() && 
           tracker.Spec.Namespace == obj.GetNamespace() {
            requests = append(requests, reconcile.Request{
                NamespacedName: types.NamespacedName{
                    Name:      tracker.Name,
                    Namespace: tracker.Namespace,
                },
            })
        }
    }
    return requests
}

func (r *DeploymentTrackerReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
    logger := log.FromContext(ctx)

    // 1. Custom Resource 가져오기
    tracker := &v1alpha1.DeploymentTracker{}
    if err := r.Get(ctx, req.NamespacedName, tracker); err != nil {
        return reconcile.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Deployment 상태 확인
    deployment := &appsv1.Deployment{}
    if err := r.Get(ctx, client.ObjectKey{
        Name:      tracker.Spec.DeploymentName,
        Namespace: tracker.Spec.Namespace,
    }, deployment); err != nil {
        logger.Error(err, "Failed to fetch Deployment")
        // Deployment를 찾을 수 없는 경우 상태 업데이트
        tracker.Status.Ready = false
        if err := r.Status().Update(ctx, tracker); err != nil {
            logger.Error(err, "Failed to update DeploymentTracker status")
        }
        return reconcile.Result{RequeueAfter: time.Second * 30}, err
    }

    // 3. 배포 상태 확인
    isReady := deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
    
    // 상태가 변경된 경우에만 처리
    if tracker.Status.Ready != isReady {
        tracker.Status.Ready = isReady
        
        if isReady {
            // 4. 이벤트 기록
            r.Recorder.Event(tracker, corev1.EventTypeNormal, "DeploymentReady",
                fmt.Sprintf("Deployment %s is running successfully", deployment.Name))

            // 5. 알림 전송
            if tracker.Spec.Notify.Slack != "" {
                if err := sendSlackNotification(tracker.Spec.Notify.Slack,
                    fmt.Sprintf("Deployment %s is successfully running", deployment.Name)); err != nil {
                    logger.Error(err, "Failed to send Slack notification")
                }
            }

            if tracker.Spec.Notify.Email != "" {
                if err := sendEmailNotification(tracker.Spec.Notify.Email,
                    fmt.Sprintf("Deployment %s is successfully running", deployment.Name)); err != nil {
                    logger.Error(err, "Failed to send Email notification")
                }
            }
        }

        // 6. 상태 업데이트
        if err := r.Status().Update(ctx, tracker); err != nil {
            logger.Error(err, "Failed to update DeploymentTracker status")
            return reconcile.Result{}, err
        }
    }

    return reconcile.Result{RequeueAfter: time.Minute}, nil
}

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

func sendEmailNotification(email, message string) error {
    // 이메일 서버 설정
    smtpHost := "smtp.gmail.com"
    smtpPort := "587"
    smtpUsername := os.Getenv("SMTP_USERNAME")
    smtpPassword := os.Getenv("SMTP_PASSWORD")

    auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

    to := []string{email}
    msg := []byte(fmt.Sprintf("Subject: Deployment Status Update\r\n\r\n%s", message))

    err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUsername, to, msg)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }

    return nil
}