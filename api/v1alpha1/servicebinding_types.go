/*
Copyright 2023 The Primaza Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceBindingSpec defines the desired state of ServiceBinding
type ServiceBindingSpec struct {

	// ServiceEndpointDefinitionSecret is the name of the secret to project into the application
	// +required
	ServiceEndpointDefinitionSecret string `json:"serviceEndpointDefinitionSecret"`

	// Application resource to inject the binding info.
	// It could be any process running within a container.
	// From the spec:
	// A Service Binding resource **MUST** define a `.spec.application`
	// which is an `ObjectReference`-like declaration to a `PodSpec`-able
	// resource.  A `ServiceBinding` **MAY** define the application
	// reference by-name or by-[label selector][ls]. A name and selector
	// **MUST NOT** be defined in the same reference.
	// +required
	Application *Application `json:"application"`

	// Env creates environment variables based on the Secret values
	Env []Environment `json:"env,omitempty"`
}

// Environment represents a key to Secret data keys and name of the environment variable
type Environment struct {
	// Name of the environment variable
	Name string `json:"name"`

	// Secret data key
	Key string `json:"key"`
}

// These are valid conditions of ServiceBinding.
const (
	// ServiceBindingReady means the ServiceBinding has projected the secret
	//  and the Workload is ready to start. It does not indicate the condition
	// of either the Service or the Workload resources referenced.
	ServiceBindingConditionReady = "Ready"
)

// ServiceBindingStatus defines the observed state of ServiceBinding.
// +k8s:openapi-gen=true
type ServiceBindingStatus struct {
	// The status of the service binding along with reason and type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Enum=Ready;Malformed
	// The state of the service binding observed
	// +kubebuilder:default:=Malformed
	State string `json:"state,omitempty"`
}

// ConditionReady specifies that the resource is ready.
// For long-running resources.
const (
	ServiceBindingStateReady     string = "Ready"
	ServiceBindingStateMalformed string = "Malformed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceBinding is the Schema for the servicebindings API
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceBindingSpec `json:"spec,omitempty"`

	// Observed status of the service binding within the namespace
	Status ServiceBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceBindingList contains a list of ServiceBinding
type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

// Application resource to inject the binding info.
// It could be any process running within a container.
type Application struct {
	// API version of the referent.
	APIVersion string `json:"apiVersion"`

	// Kind of the referent.
	Kind string `json:"kind"`

	// Name of the referent.
	// Mutually exclusive with Selector.
	// +optional
	Name string `json:"name,omitempty"`

	// Selector of the referents.
	// Mutually exclusive with Name.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	Containers []intstr.IntOrString `json:"containers,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ServiceBinding{}, &ServiceBindingList{})
}

func (sb *ServiceBinding) HasDeletionTimestamp() bool {
	return !sb.DeletionTimestamp.IsZero()
}

func (sb *ServiceBinding) GetSpec() interface{} {
	return &sb.Spec
}
