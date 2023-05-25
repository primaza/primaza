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
	"errors"
	"fmt"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func DeleteAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string, deploymentName string, configMapName string) error {
	errs := []error{}
	if err := cli.AppsV1().Deployments(namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("error deleting deployment '%s/%s': %w", namespace, deploymentName, err))
	}

	if err := cli.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("error deleting configmap '%s/%s': %w", namespace, configMapName, err))
	}

	return errors.Join(errs...)
}

func PushAgent(
	ctx context.Context,
	cli *kubernetes.Clientset,
	namespace string,
	ceName string,
	agentManifest string,
	image string,
	configManifest string,
	strategy primazaiov1alpha1.SynchronizationStrategy,
) error {
	errs := []error{}
	if err := createAgentDeployment(ctx, cli, namespace, ceName, agentManifest, image); err != nil && !apierrors.IsAlreadyExists(err) {
		errs = append(errs, err)
	}

	if err := createAgentConfigMap(ctx, cli, namespace, ceName, configManifest, strategy); err != nil && !apierrors.IsAlreadyExists(err) {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func createAgentConfigMap(
	ctx context.Context,
	cli *kubernetes.Clientset,
	namespace string,
	ceName string,
	configMapManifest string,
	strategy primazaiov1alpha1.SynchronizationStrategy,
) error {
	var cm corev1.ConfigMap
	err := yaml.Unmarshal([]byte(configMapManifest), &cm)
	if err != nil {
		return fmt.Errorf("unmarshal deployment error: %w", err)
	}

	cm.ObjectMeta.Namespace = namespace
	if cm.ObjectMeta.Labels == nil {
		cm.ObjectMeta.Labels = map[string]string{}
	}
	cm.ObjectMeta.Labels[constants.PrimazaClusterEnvironmentLabel] = ceName
	cm.Data["synchronization-strategy"] = string(strategy)
	if _, err := cli.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("error creating deployment: %w", err)
	}

	return nil
}

func createAgentDeployment(ctx context.Context, cli *kubernetes.Clientset, namespace string, ceName string, agentManifest string, image string) error {
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
