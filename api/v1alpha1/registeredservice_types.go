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

// RegisteredServiceConstraints defines constrains to be honored when determining
// whether the service can be claimed from certain environments.
type RegisteredServiceConstraints struct {
	// Environments defines in which environments the RegisteredService may be used.
	Environments []string `json:"environments,omitempty"`
}

// HealthCheckContainer defines the container information to be used to
// run helth checks for the service.
type HealthCheckContainer struct {
    // Container image with the client to run the test
    Image string `json:"image"`
    // Command to execute in the container to run the test
    Command string `json:"command"`

}

// RegisteredServiceHealthCheck defines metadata that can be used check
// the health of a service and report status.
type RegisteredServiceHealthCheck struct {
    // Container defines a container that will run a check against the 
    // ServiceEndpointDefinition to determine connectivity and access.
    Container HealthCheckContainer `json:"container"`
}

// ServiceClassIdentityItem defines an attribute that is necessary to
// identify a service class.
type ServiceClassIdentityItem struct {
    // Name of the service class identity attribute.
    Name string `json:"name"`

    // Value of the service class identity attribute.
    Value string `json:"value"`
}

// ServiceEndpointDefinitionSecretRef defines a reference to
// one of the keys of a secret. This reference can then be used
// when defining a ServiceEndpointDefinitionItem 
type ServiceEndpointDefinitionSecretRef struct {
    // Name of the secret reference
    Name string `json:"name"`

    // Key of the secret reference field
    Key string `json:"key"`
}

// ServiceEndpointDefinitionItem defines an attribute that is necessary for
// a client to connect to a service
type ServiceEndpointDefinitionItem struct {
    // Name of the service endpoint definition attribute.
    Name string `json:"name"`

    // Value of the service endpoint definition attribute. It is mutually
    // exclusive with ValueFromSecret.
    Value string `json:"value,omitempty"`

    // Value reference of the service endpoint definition attribute. It is mutually
    // exclusive with Value
    ValueFromSecret ServiceEndpointDefinitionSecretRef `json:"valueFromSecret,omitempty"`
}

// RegisteredServiceSpec defines the desired state of RegisteredService
type RegisteredServiceSpec struct {
	// Constraints defines under which circumstances the RegisteredService may
	// be used.
	Constraints RegisteredServiceConstraints `json:"constraints,omitempty"`

	// HealthCheck defines a health check for the underlying service.
    HealthCheck  RegisteredServiceHealthCheck `json:"healthcheck,omitempty"`

	// SLA defines the support level for this service.
	SLA string `json:"sla,omitempty"`

    // ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service. 
    ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`

    // ServiceEndpointDefinition defines a set of attributes sufficient for a
	// client to establish a connection to the service.
    ServiceEndpointDefinition []ServiceEndpointDefinitionItem `json:"serviceEndpointDefinition"`
}

// RegisteredServiceStatus defines the observed state of RegisteredService.
type RegisteredServiceStatus struct {
	// State describes the current state of the service.
	State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RegisteredService is the Schema for the registeredservices API.
type RegisteredService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegisteredServiceSpec   `json:"spec,omitempty"`
	Status RegisteredServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RegisteredServiceList contains a list of RegisteredService.
type RegisteredServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegisteredService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegisteredService{}, &RegisteredServiceList{})
}
