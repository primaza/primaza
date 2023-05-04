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
	_ "embed"
	"fmt"

	"github.com/primaza/primaza/pkg/primaza/constants"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
)

//go:embed templates/agentsvc.yaml
var agentSvcDeployment string

func DeleteServiceAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string) error {
	s := runtime.NewScheme()
	if err := appsv1.AddToScheme(s); err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}
	decode := serializer.NewCodecFactory(s).UniversalDeserializer().Decode

	obj, _, err := decode([]byte(agentSvcDeployment), nil, nil)
	if err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}

	dep := obj.(*appsv1.Deployment)
	if err := cli.AppsV1().Deployments(namespace).Delete(ctx, dep.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting deployment: %w", err)
	}

	return nil
}

func PushServiceAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string, ceName string, image string) error {
	if err := createAgentSvcDeployment(ctx, cli, namespace, ceName, image); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createAgentSvcDeployment(ctx context.Context, cli *kubernetes.Clientset, namespace string, ceName string, image string) error {
	s := runtime.NewScheme()
	if err := appsv1.AddToScheme(s); err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}
	decode := serializer.NewCodecFactory(s).UniversalDeserializer().Decode

	obj, _, err := decode([]byte(agentSvcDeployment), nil, nil)
	if err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}

	dep := obj.(*appsv1.Deployment)
	dep.ObjectMeta.Namespace = namespace
	dep.Spec.Template.Spec.Containers[0].Image = image
	dep.ObjectMeta.Labels[constants.PrimazaClusterEnvironmentLabel] = ceName
	if _, err := cli.AppsV1().Deployments(namespace).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("error creating deployment: %w", err)
	}
	return nil
}
