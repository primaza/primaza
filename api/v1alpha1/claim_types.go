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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClaimState string

const (
	ClaimConditionReady ClaimState = "Ready"
	ClaimStatePending   ClaimState = "Pending"
	ClaimStateResolved  ClaimState = "Resolved"
	ClaimStateInvalid   ClaimState = "Invalid"
)

// ClaimSpec defines the desired state of Claim
type ClaimSpec struct {
	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A Claim whose ServiceClassIdentity
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

	// Envs allows projecting Service Endpoint Definition's data as Environment Variables in the Pod
	// +optional
	Envs []Environment `json:"envs,omitempty"`
}

type ClaimApplicationClusterContext struct {
	ClusterEnvironmentName string `json:"clusterEnvironmentName"`
	Namespace              string `json:"namespace"`
}

// ClaimStatus defines the observed state of Claim
type ClaimStatus struct {
	// The state of the Claim observed
	//+kubebuilder:validation:Enum=Pending;Resolved;Invalid
	//+kubebuilder:default:=Pending
	State ClaimState `json:"state"`
	// Unique ID For the Claim
	ClaimID string `json:"claimID,omitempty"`
	// Claimed RegisteredService Info
	RegisteredService *corev1.ObjectReference `json:"registeredService,omitempty"`
	// The status of the service binding along with reason and type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of the Claim"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Claim is the Schema for the claims API
type Claim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClaimSpec   `json:"spec,omitempty"`
	Status ClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClaimList contains a list of Claim
type ClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Claim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Claim{}, &ClaimList{})
}

func (sc *Claim) HasDeletionTimestamp() bool {
	return !sc.DeletionTimestamp.IsZero()
}
