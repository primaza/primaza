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

type ApplicationServiceClaimState string

const (
	ApplicationServiceClaimStateReady    ApplicationServiceClaimState = "Ready"
	ApplicationServiceClaimStatePending  ApplicationServiceClaimState = "Pending"
	ApplicationServiceClaimStateResolved ApplicationServiceClaimState = "Resolved"
	ApplicationServiceClaimStateInvalid  ApplicationServiceClaimState = "Invalid"
)

// ApplicationServiceClaimSpec defines the desired state of ApplicationServiceClaim
type ApplicationServiceClaimSpec struct {
	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ControlPlaneServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`
	// ServiceEndpointDefinition defines a set of attributes sufficient for a
	// client to establish a connection to the service.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	ServiceEndpointDefinitionKeys []string `json:"serviceEndpointDefinitionKeys"`
	// Rules to match workloads to bind
	// +required
	Application ApplicationSelector `json:"application"`
	// Control Plane Service Claim target
	Target *ControlPlaneServiceClaimTarget `json:"target,omitempty"`
	// Envs allows projecting Service Endpoint Definition's data as Environment Variables in the Pod
	// +optional
	Envs []Environment `json:"envs,omitempty"`
}

// The Application Service Claim target.
// It can be an entire environment or a single application
// +kubebuilder:validation:MaxProperties:=1
// +kubebuilder:validation:MinProperties:=1
type ApplicationServiceClaimTarget struct {
	// EnvironmentTag allows the controller to search for those application cluster
	// environments that define such EnvironmentTag
	// +optional
	EnvironmentTag string `json:"environmentTag,omitempty"`
	// +optional
	ApplicationClusterContext *ServiceClaimApplicationClusterContext `json:"applicationClusterContext,omitempty"`
}

type ApplicationServiceClaimApplicationClusterContext struct {
	ClusterEnvironmentName string `json:"clusterEnvironmentName"`
	Namespace              string `json:"namespace"`
}

// ApplicationServiceClaimStatus defines the observed state of ApplicationServiceClaim
type ApplicationServiceClaimStatus struct {
	// The state of the ControlPlaneServiceClaim observed
	//+kubebuilder:validation:Enum=Pending;Resolved;Invalid
	//+kubebuilder:default:=Pending
	State ControlPlaneServiceClaimState `json:"state"`
	// Unique ID For the ServiceClaim
	ClaimID string `json:"claimID,omitempty"`
	// Claimed RegisteredService Info
	RegisteredService *corev1.ObjectReference `json:"registeredService,omitempty"`
	// The status of the service binding along with reason and type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of the ServiceClaim"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationServiceClaim is the Schema for the applicationserviceclaims API
type ApplicationServiceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneServiceClaimSpec   `json:"spec,omitempty"`
	Status ControlPlaneServiceClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationServiceClaimList contains a list of ApplicationServiceClaim
type ApplicationServiceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlaneServiceClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationServiceClaim{}, &ApplicationServiceClaimList{})
}

func (sc *ApplicationServiceClaim) HasDeletionTimestamp() bool {
	return !sc.DeletionTimestamp.IsZero()
}
