// api/v1alpha1/groupversion_info.go
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{
		Group:   "ddukbg.k8s",
		Version: "v1alpha1",
	}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Required for pkg/runtime.Object interface
var _ runtime.Object = &DeploymentTracker{}
var _ runtime.Object = &DeploymentTrackerList{}
