//go:build !ignore_autogenerated

/*
Copyright 2024 ddukbg.

Licensed under the MIT License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageState) DeepCopyInto(out *ImageState) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageState.
func (in *ImageState) DeepCopy() *ImageState {
	if in == nil {
		return nil
	}
	out := new(ImageState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifyConfig) DeepCopyInto(out *NotifyConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifyConfig.
func (in *NotifyConfig) DeepCopy() *NotifyConfig {
	if in == nil {
		return nil
	}
	out := new(NotifyConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceState) DeepCopyInto(out *ResourceState) {
	*out = *in
	out.ImageState = in.ImageState
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceState.
func (in *ResourceState) DeepCopy() *ResourceState {
	if in == nil {
		return nil
	}
	out := new(ResourceState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTarget) DeepCopyInto(out *ResourceTarget) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTarget.
func (in *ResourceTarget) DeepCopy() *ResourceTarget {
	if in == nil {
		return nil
	}
	out := new(ResourceTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTracker) DeepCopyInto(out *ResourceTracker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTracker.
func (in *ResourceTracker) DeepCopy() *ResourceTracker {
	if in == nil {
		return nil
	}
	out := new(ResourceTracker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceTracker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTrackerList) DeepCopyInto(out *ResourceTrackerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ResourceTracker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTrackerList.
func (in *ResourceTrackerList) DeepCopy() *ResourceTrackerList {
	if in == nil {
		return nil
	}
	out := new(ResourceTrackerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceTrackerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTrackerSpec) DeepCopyInto(out *ResourceTrackerSpec) {
	*out = *in
	out.Target = in.Target
	out.Notify = in.Notify
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTrackerSpec.
func (in *ResourceTrackerSpec) DeepCopy() *ResourceTrackerSpec {
	if in == nil {
		return nil
	}
	out := new(ResourceTrackerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTrackerStatus) DeepCopyInto(out *ResourceTrackerStatus) {
	*out = *in
	if in.LastUpdated != nil {
		in, out := &in.LastUpdated, &out.LastUpdated
		*out = (*in).DeepCopy()
	}
	out.CurrentState = in.CurrentState
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTrackerStatus.
func (in *ResourceTrackerStatus) DeepCopy() *ResourceTrackerStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceTrackerStatus)
	in.DeepCopyInto(out)
	return out
}