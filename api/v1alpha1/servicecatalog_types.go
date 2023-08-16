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

type ServiceCatalogServiceByLabel struct {
	// ServiceCatalogService defines the service that is claimed by labels.
	ServiceCatalogService `json:",inline"`

	// Labels labels selector for the service
	Labels *metav1.LabelSelector `json:"labels"`
}

type ServiceCatalogService struct {
	// Name defines the name of the known service
	Name string `json:"name"`

	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`

	// ServiceEndpointDefinitionKeys defines a set of keys listing the
	// information this service provides to a workload.
	ServiceEndpointDefinitionKeys []string `json:"serviceEndpointDefinitionKeys"`
}

// ServiceCatalogSpec defines the desired state of ServiceCatalog
type ServiceCatalogSpec struct {
	// Services contains a list of services that are known to Primaza.
	Services []ServiceCatalogService `json:"services,omitempty"`
	// ClaimedByLabels contains a list of services that are claimed by labels.
	ClaimedByLabels []ServiceCatalogServiceByLabel `json:"claimedByLabels,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceCatalog is the Schema for the servicecatalogs API
type ServiceCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceCatalogSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceCatalogList contains a list of ServiceCatalog
type ServiceCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceCatalog{}, &ServiceCatalogList{})
}
