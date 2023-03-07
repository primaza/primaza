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
)

type ServiceClassMapping struct {
	// Name of the data referred to
	Name string `json:"name"`

	// JsonPath defines where data lives in the service resource.  This query
	// must resolve to a single value (e.g. not an array of values).
	JsonPath string `json:"jsonPath"`
}

// ServiceClassResource defines
type ServiceClassResource struct {
	// APIVersion of the underlying service resource
	APIVersion string `json:"apiVersion"`

	// Kind of the underlying service resource
	Kind string `json:"kind"`

	// ServiceEndpointDefinitionMapping defines how a key-value mapping projected
	// into services may be constructed.
	ServiceEndpointDefinitionMapping []ServiceClassMapping `json:"serviceEndpointDefinitionMapping"`
}

// ServiceClassSpec defines the desired state of ServiceClass
type ServiceClassSpec struct {
	// Constraints defines under which circumstances the ServiceClass may
	// be used.
	// +optional
	Constraints *EnvironmentConstraints `json:"constraints,omitempty"`

	// Resource defines the resource type to be used to convert into Registered
	// Services
	Resource ServiceClassResource `json:"resource"`

	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`
}

// ServiceClassStatus defines the observed state of ServiceClass
type ServiceClassStatus struct {
	Conditions metav1.ConditionStatus `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceClass is the Schema for the serviceclasses API
type ServiceClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceClassSpec   `json:"spec,omitempty"`
	Status ServiceClassStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceClassList contains a list of ServiceClass
type ServiceClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceClass{}, &ServiceClassList{})
}
