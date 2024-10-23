package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// GroupVersion is group version used to register these objects
var GroupVersion = scheme.GroupVersion{
	Group:   "ddukbg",
	Version: "v1alpha1",
}

// SchemeBuilder adds the scheme to the manager
var SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
