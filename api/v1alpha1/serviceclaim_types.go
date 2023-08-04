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

type ServiceClaimState string

const (
	ServiceClaimConditionReady ServiceClaimState = "Ready"
	ServiceClaimStatePending   ServiceClaimState = "Pending"
	ServiceClaimStateResolved  ServiceClaimState = "Resolved"
	ServiceClaimStateInvalid   ServiceClaimState = "Invalid"
)

// ServiceClaimSpec defines the desired state of ServiceClaim
type ServiceClaimSpec struct {
	// ServiceClassIdentity defines a set of attributes that are sufficient to
	// identify a service class.  A ServiceClaim whose ServiceClassIdentity
	// field is a subset of a RegisteredService's keys can claim that service.
	// +required
	ServiceClassIdentity []ServiceClassIdentityItem `json:"serviceClassIdentity"`
	// ServiceEndpointDefinition defines a set of attributes sufficient for a
	// client to establish a connection to the service.
	// +required
	ServiceEndpointDefinitionKeys []string `json:"serviceEndpointDefinitionKeys"`
	// Rules to match workloads to bind
	// +required
	Application ServiceClaimApplicationSelector `json:"application"`
	// Service Claim target
	Target *ServiceClaimTarget `json:"target,omitempty"`
	// Envs allows projecting Service Endpoint Definition's data as Environment Variables in the Pod
	// +optional
	Envs []Environment `json:"envs,omitempty"`
}

// Application resource to inject the binding info.
// It could be any process running within a container.
type ServiceClaimApplicationSelector struct {
	// API version of the referent.
	//+required
	APIVersion string `json:"apiVersion"`
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	//+required
	Kind string `json:"kind"`
	// Rules to match a resource
	//+required
	Selector ApplicationMatcher `json:"selector"`
}

// +kubebuilder:validation:MaxProperties:=1
// +kubebuilder:validation:MinProperties:=1
// Express the rules to match a resource by name or by labels
type ApplicationMatcher struct {
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	//+optional
	ByName string `json:"byName,omitempty"`
	// Selector is a query that selects the workload or workloads to bind the service to
	//+optional
	ByLabels *metav1.LabelSelector `json:"byLabels,omitempty"`
}

// +kubebuilder:validation:MaxProperties:=1
// +kubebuilder:validation:MinProperties:=1
// The Service Claim target.
// It can be an entire environment or a single application
type ServiceClaimTarget struct {
	// EnvironmentTag allows the controller to search for those application cluster
	// environments that define such EnvironmentTag
	// +optional
	EnvironmentTag string `json:"environmentTag,omitempty"`
	// +optional
	ApplicationClusterContext *ServiceClaimApplicationClusterContext `json:"applicationClusterContext,omitempty"`
}

type ServiceClaimApplicationClusterContext struct {
	ClusterEnvironmentName string `json:"clusterEnvironmentName"`
	Namespace              string `json:"namespace"`
}

// ServiceClaimStatus defines the observed state of ServiceClaim
type ServiceClaimStatus struct {
	// The state of the ServiceClaim observed
	//+kubebuilder:validation:Enum=Pending;Resolved;Invalid
	//+kubebuilder:default:=Pending
	State ServiceClaimState `json:"state"`
	// Unique ID For the ServiceClaim
	ClaimID string `json:"claimID,omitempty"`
	// Claimed RegisteredService Info
	RegisteredService *corev1.ObjectReference `json:"registeredService,omitempty"`
	// The status of the service binding along with reason and type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Original ServiceClassIdentity as it was at ServiceClaim creation time
	OriginalServiceClassIdentity []ServiceClassIdentityItem `json:"originalServiceClassIdentity,omitempty"`
	// Original ServiceEndpointDefinitionKeys as they were at ServiceClaim creation time
	OriginalServiceEndpointDefinitionKeys []string `json:"originalServiceEndpointDefinitionKeys,omitempty"`
	// Original Service Claim target
	OriginalTarget *ServiceClaimTarget `json:"originalTarget,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of the ServiceClaim"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
