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

type ServiceEndpointDefinitionMappings struct {
	ResourceFields  []ServiceClassResourceFieldMapping  `json:"resourceFields,omitempty"`
	SecretRefFields []ServiceClassSecretRefFieldMapping `json:"secretRefFields,omitempty"`
}

type ServiceClassResourceFieldMapping struct {
	// Name of the data referred to
	Name string `json:"name"`

	// JsonPath defines where data lives in the service resource.  This query
	// must resolve to a single value (e.g. not an array of values).
	JsonPath string `json:"jsonPath"`

	// Secret indicates whether or not the mapping data needs to be stored in a secret.
	// +optional
	// +kubebuilder:default=true
	Secret bool `json:"secret"`
}

type ServiceClassSecretRefFieldMapping struct {
	// Name of the data referred to
	Name string `json:"name"`

	// SecretName defines a constant value or a JsonPath used to extract from
	// resource's specification the name of a linked secret
	SecretName FieldMapping `json:"secretName"`

	// SecretKey defines a constant value or a JsonPath used to extract from
	// resource's specification the Key to be copied from the linked secret
	SecretKey FieldMapping `json:"secretKey"`
}

// +kubebuilder:validation:MaxProperties:=1
// +kubebuilder:validation:MinProperties:=1
type FieldMapping struct {
	// Constant is a constant value for the field
	Constant *string `json:"constant,omitempty"`
	// JsonPathExpr represents a jsonPath for extracting the field
	JsonPathExpr *string `json:"jsonPath,omitempty"`
}

// ServiceClassResource defines
type ServiceClassResource struct {
	// APIVersion of the underlying service resource
	APIVersion string `json:"apiVersion"`

	// Kind of the underlying service resource
	Kind string `json:"kind"`

	// ServiceEndpointDefinitionMappings defines how a key-value mapping projected
	// into services may be constructed.
	ServiceEndpointDefinitionMappings ServiceEndpointDefinitionMappings `json:"serviceEndpointDefinitionMappings"`
}

// ServiceClassSpec defines the desired state of ServiceClass
type ServiceClassSpec struct {
	// Constraints defines under which circumstances the ServiceClass may
	// be used.
	// +optional
	Constraints *EnvironmentConstraints `json:"constraints,omitempty"`

	// HealthCheck sets the default health check for generated registered services
	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`

	// Resource defines the resource type to be used to convert into Registered
	// Services
	Resource ServiceClassResource `json:"resource"`

	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`
}

func (s ServiceClassSpec) GetEnvironmentConstraints() []string {
	if s.Constraints != nil {
		return s.Constraints.Environments
	}

	return nil
}

// ServiceClassStatus defines the observed state of ServiceClass
type ServiceClassStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
