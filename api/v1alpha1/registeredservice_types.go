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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RegisteredServiceSpec defines the desired state of RegisteredService
type RegisteredServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of RegisteredService. Edit registeredservice_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// RegisteredServiceStatus defines the observed state of RegisteredService
type RegisteredServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
