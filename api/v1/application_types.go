/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application.
type ApplicationSpec struct {
	// Replicas is the desired number of pods managed by this Application.
	// +optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Template describes the pods that will be created.
	Template ApplicationTemplate `json:"template"`
}

// ApplicationTemplate describes the pods that will be created.
// Only the minimum fields needed by this operator are exposed to keep the
// generated CRD small; expand the struct (or add kubebuilder markers) when
// more pod knobs are required.
type ApplicationTemplate struct {
	// Containers is the list of application containers.
	// +listType=map
	// +listMapKey=name
	Containers []ApplicationContainer `json:"containers"`
}

// ApplicationContainer is a slimmed-down container definition.
type ApplicationContainer struct {
	// Name must be unique within the pod.
	Name string `json:"name"`

	// Image is the container image (e.g. nginx:1.25).
	Image string `json:"image"`

	// Entrypoint array. Not executed within a shell.
	// The container image's ENTRYPOINT is used if this is not provided.
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint.
	// The container image's CMD is used if this is not provided.
	// +optional
	Args []string `json:"args,omitempty"`

	// ImagePullPolicy defines how the image is pulled.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if the :latest tag is specified, or IfNotPresent otherwise.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Ports exposed by the container.
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// Env is the list of environment variables to set in the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom is the list of sources to populate environment variables in the container.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Resources describes the compute resource requirements (requests and limits).
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// LivenessProbe checks if the container is alive.
	// If not provided, no liveness probe is configured.
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe checks if the container is ready to serve traffic.
	// If not provided, no readiness probe is configured.
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// Conditions represent the latest available observations of the
	// Application's current state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Replicas is the total number of non-terminated pods targeted by this
	// Application (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas is the number of pods targeted by this Application with a
	// Ready condition.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Application
	// +required
	Spec ApplicationSpec `json:"spec"`

	// status defines the observed state of Application
	// +optional
	Status ApplicationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
