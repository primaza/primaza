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

type ServiceClaimApplicationClusterContext struct {
	ClusterEnvironmentName string `json:"clusterEnvironmentName"`
	Namespace              string `json:"namespace"`
}

// ServiceClaimSpec defines the desired state of ServiceClaim
type ServiceClaimSpec struct {
	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`

	// ServiceEndpointDefinition defines a set of attributes sufficient for a
	// client to establish a connection to the service.
	ServiceEndpointDefinitionKeys []string `json:"serviceEndpointDefinitionKeys"`

	Application ApplicationSelector `json:"application,omitempty"`
	// EnvironmentTag allows the controller to search for those application cluster
	// environments that define such EnvironmentTag
	// +optional
	EnvironmentTag string `json:"environmentTag,omitempty"`
	// +optional
	ApplicationClusterContext *ServiceClaimApplicationClusterContext `json:"applicationClusterContext,omitempty"`
}

// ServiceClaimStatus defines the observed state of ServiceClaim
type ServiceClaimStatus struct {
	//+kubebuilder:validation:Enum=Pending;Resolved;Invalid
	//+kubebuilder:default:=Pending
	State             ServiceClaimState  `json:"state"`
	ClaimID           string             `json:"claimID,omitempty"`
	RegisteredService string             `json:"registeredService"`
	Conditions        []metav1.Condition `json:"conditions,omitempty"`
}

type ServiceClaimState string

const (
	ServiceClaimStatePending  ServiceClaimState = "Pending"
	ServiceClaimStateResolved ServiceClaimState = "Resolved"
	ServiceClaimStateInvalid  ServiceClaimState = "Invalid"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceClaim is the Schema for the serviceclaims API
type ServiceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceClaimSpec   `json:"spec,omitempty"`
	Status ServiceClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceClaimList contains a list of ServiceClaim
type ServiceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceClaim{}, &ServiceClaimList{})
}

func (sc *ServiceClaim) HasDeletionTimestamp() bool {
	return !sc.DeletionTimestamp.IsZero()
}
