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

package workercluster

import (
	"context"
	"fmt"

	"github.com/primaza/primaza/pkg/primaza/constants"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func DeleteApplicationAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string) error {
	if err := cli.AppsV1().Deployments(namespace).Delete(ctx, constants.ApplicationAgentDeploymentName, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting deployment: %w", err)
	}

	return nil
}

func PushApplicationAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string, ceName string, agentManifest string, image string) error {
	if err := createAgentAppDeployment(ctx, cli, namespace, ceName, agentManifest, image); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createAgentAppDeployment(ctx context.Context, cli *kubernetes.Clientset, namespace string, ceName string, agentManifest string, image string) error {
	var dep appsv1.Deployment
	err := yaml.Unmarshal([]byte(agentManifest), &dep)
	if err != nil {
		return fmt.Errorf("unmarshal deployment error: %w", err)
	}
	dep.ObjectMeta.Namespace = namespace
	dep.Spec.Template.Spec.Containers[0].Image = image
	dep.ObjectMeta.Labels[constants.PrimazaClusterEnvironmentLabel] = ceName
	if _, err := cli.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("error creating deployment: %w", err)
	}

	return nil
}
