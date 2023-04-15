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

package constants

const (
	PrimazaNamespace               = "primaza-system"
	ServiceAgentDeploymentName     = "primaza-svc-agent"
	ApplicationAgentDeploymentName = "primaza-app-agent"
	// This is the name of the secret that contains the information the service
	// agents needs to write back registered services up to primaza.  It contains
	// two keys: `kubeconfig`, a serialized kubeconfig for the upstream kubeconfig
	// cluster, and `namespace`, the namespace to write registered services to
	ApplicationAgentKubeconfigSecretName = "kubeconfig-primaza-app" // #nosec G101
	ServiceAgentKubeconfigSecretName     = "kubeconfig-primaza-svc" // #nosec G101
)
