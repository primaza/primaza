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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ServiceClassIdentityItem defines an attribute that is necessary to
// identify a service class.
type ServiceClassIdentityItem struct {
	// Name of the service class identity attribute.
	Name string `json:"name"`

	// Value of the service class identity attribute.
	Value string `json:"value"`
}

// Application resource to inject the binding info.
// It could be any process running within a container.
// +kubebuilder:validation:XValidation:rule="!(has(self.name) && has(self.selector))",message="`name` and `selector` can not be used at the same time"
// +kubebuilder:validation:XValidation:rule="has(self.name) || has(self.selector)",message="one among `name` and `selector` is required"
type ApplicationSelector struct {
	// API version of the referent.
	//+required
	APIVersion string `json:"apiVersion"`
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	//+required
	Kind string `json:"kind"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	//+optional
	Name string `json:"name,omitempty"`
	// Selector is a query that selects the workload or workloads to bind the service to
	//+optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// EnvironmentConstraints defines the constraints on environment for which
// the resource may be used.
type EnvironmentConstraints struct {
	// Environments defines the environments that the RegisteredService may be
	// used in.
	Environments []string `json:"environments,omitempty"`
}

// Environment represents a key to Secret data keys and name of the environment variable
type Environment struct {
	// Name of the environment variable
	Name string `json:"name"`

	// Secret data key
	Key string `json:"key"`
}

// HealthCheckContainer defines the container information to be used to
// run helth checks for the service.
type HealthCheckContainer struct {
	// Container image with the client to run the test
	Image string `json:"image"`
	// Command to execute in the container to run the test
	Command []string `json:"command"`
	// Minutes between runs of the healthcheck container.  Must be greater than
	// or equal to 1.
	//+optional
	//+kubebuilder:default:=5
	//+kubebuilder:validation:Minimum:=1
	Minutes uint32 `json:"minutes"`
}

// HealthCheck defines metadata that can be used check
// the health of a service and report status.
type HealthCheck struct {
	// Container defines a container that will run a check against the
	// ServiceEndpointDefinition to determine connectivity and access.
	Container HealthCheckContainer `json:"container"`
}
