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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SynchronizationStrategyPull SynchronizationStrategy = "Pull"
	SynchronizationStrategyPush SynchronizationStrategy = "Push"
)

var synchronizationStrategies = map[string]SynchronizationStrategy{
	string(SynchronizationStrategyPull): SynchronizationStrategyPull,
	string(SynchronizationStrategyPush): SynchronizationStrategyPush,
}

func ParseSynchronizationStrategy(s string) (*SynchronizationStrategy, error) {
	if p, ok := synchronizationStrategies[s]; ok {
		return &p, nil
	}
	return nil, fmt.Errorf("cannot parse '%s' as SynchronizationStrategy", s)
}

// SynchronizationStrategy defines the synchronization strategy
type SynchronizationStrategy string

// ClusterEnvironmentSpec defines the desired state of ClusterEnvironment
type ClusterEnvironmentSpec struct {
	// The environment associated to the ClusterEnvironment instance
	EnvironmentName string `json:"environmentName"`

	// Name of the Secret where connection (kubeconfig) information to target cluster is stored
	ClusterContextSecret string `json:"clusterContextSecret"`

	// Description of the ClusterEnvironment
	Description string `json:"description,omitempty"`

	// Labels
	Labels []string `json:"labels,omitempty"`

	// Namespaces in target cluster where applications are deployed
	ApplicationNamespaces []string `json:"applicationNamespaces,omitempty"`

	// Namespaces in target cluster where services are discovered
	ServiceNamespaces []string `json:"serviceNamespaces,omitempty"`

	// Cluster Admin's contact information
	ContactInfo string `json:"contactInfo,omitempty"`

	// SynchronizationStrategy defines whether Primaza will watch clusters (Pull)
	// or Agents will Push data as they have (Push)
	//+kubebuilder:validation:Enum=Pull;Push
	//+kubebuilder:default:=Push
	SynchronizationStrategy SynchronizationStrategy `json:"synchronizationStrategy"`
}

// ClusterEnvironmentStatus defines the observed state of ClusterEnvironment
type ClusterEnvironmentStatus struct {
	// The State of the cluster environment
	//+kubebuilder:validation:Enum=Online;Offline;Partial
	//+kubebuilder:default:=Offline
	State ClusterEnvironmentState `json:"state"`

	// Status Conditions
	Conditions []metav1.Condition `json:"conditions"`
}

type ClusterEnvironmentState string

const (
	ClusterEnvironmentStateOnline  ClusterEnvironmentState = "Online"
	ClusterEnvironmentStatePartial ClusterEnvironmentState = "Partial"
	ClusterEnvironmentStateOffline ClusterEnvironmentState = "Offline"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Environment",type="string",JSONPath=".spec.environmentName",description="the environment associated to the ClusterEnvironment instance"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of the ClusterEnvironment"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterEnvironment is the Schema for the clusterenvironments API
type ClusterEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterEnvironmentSpec   `json:"spec,omitempty"`
	Status ClusterEnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterEnvironmentList contains a list of ClusterEnvironment
type ClusterEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterEnvironment{}, &ClusterEnvironmentList{})
}

func (ce *ClusterEnvironment) HasDeletionTimestamp() bool {
	return !ce.DeletionTimestamp.IsZero()
}
