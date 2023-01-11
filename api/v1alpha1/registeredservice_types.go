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

type RegisteredServiceConstraints struct {
    // Environments defines in which environments the RegisteredService may be used.
    Environments []string `json:"environments,omitempty"`
}

// RegisteredServiceSpec defines the desired state of RegisteredService
type RegisteredServiceSpec struct {
    // Constraints defines under which circumstances the RegisteredService may
    // be used.
    Constraints RegisteredServiceConstraints `json:"constraints,omitempty"`

    // HealthCheck defines a health check for the underlying service.
    // HealthCheck 

    // SLA defines the support level for this service.
    SLA string `json:"sla,omitempty"`

    // ServiceClassIdentity defines a set of attributes that are sufficient to
    // identify a service class.  A ServiceClaim whose ServiceClassIdentity
    // field is a subset of a RegisteredService's keys can claim that service. 
    ServiceClassIdentity []string `json:"serviceClassIdentity"`

    // ServiceEndpointDefinition defines a set of attributes sufficient for a
    // client to establish a connection to the service.
    ServiceEndpointDefinition map[string]string `json:"serviceEndpointDefinition"`
}

// RegisteredServiceStatus defines the observed state of RegisteredService
type RegisteredServiceStatus struct {
    // State describes the current state of the service.
    State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RegisteredService is the Schema for the registeredservices API
type RegisteredService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegisteredServiceSpec   `json:"spec,omitempty"`
	Status RegisteredServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RegisteredServiceList contains a list of RegisteredService
type RegisteredServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegisteredService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegisteredService{}, &RegisteredServiceList{})
}
