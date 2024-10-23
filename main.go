package main

import (
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrl "sigs.k8s.io/controller-runtime"
	"k8s-deploy-watcher/controllers"
	"k8s-deploy-watcher/api/v1alpha1"
)

func main() {
	// Manager 설정
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: v1alpha1.SchemeBuilder.AddToScheme,
	})
	if err != nil {
		os.Exit(1)
	}

	// Reconciler 등록
	if err := (&controllers.DeploymentTrackerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}

	// Operator 실행
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
