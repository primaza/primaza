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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
)

func DeleteApplicationAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string) error {
	s := runtime.NewScheme()
	if err := appsv1.AddToScheme(s); err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}

	decode := serializer.NewCodecFactory(s).UniversalDeserializer().Decode

	obj, _, err := decode([]byte(agentAppDeployment), nil, nil)
	if err != nil {
		return fmt.Errorf("decoder error: %w", err)
	}

	dep := obj.(*appsv1.Deployment)
	if err := cli.AppsV1().Deployments(namespace).Delete(ctx, dep.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting deployment: %w", err)
	}

	return nil
}

func PushApplicationAgent(ctx context.Context, cli *kubernetes.Clientset, namespace string) error {
	if _, err := createAgentAppDeployment(ctx, cli, namespace); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createAgentAppDeployment(ctx context.Context, cli *kubernetes.Clientset, namespace string) (*appsv1.Deployment, error) {
	s := runtime.NewScheme()
	if err := appsv1.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("decoder error: %w", err)
	}
	decode := serializer.NewCodecFactory(s).UniversalDeserializer().Decode

	obj, _, err := decode([]byte(agentAppDeployment), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decoder error: %w", err)
	}

	dep := obj.(*appsv1.Deployment)
	r, err := cli.AppsV1().Deployments(namespace).Create(ctx, dep, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating deployment: %w", err)
	}

	return r, nil
}

const agentAppDeployment string = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: primaza-controller-agentapp
  labels:
    control-plane: primaza-controller-agentapp
    app.kubernetes.io/name: deployment
    app.kubernetes.io/instance: primaza-controller-agentapp
    app.kubernetes.io/component: agentapp-manager
    app.kubernetes.io/created-by: primaza
    app.kubernetes.io/part-of: primaza
spec:
  selector:
    matchLabels:
      control-plane: primaza-controller-agentapp
  replicas: 1
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: primaza-controller-agentapp
    spec:
      securityContext:
        runAsNonRoot: true
      containers:
      - command:
        - /manager
        args:
        - --leader-elect
        image: agentapp:latest
        imagePullPolicy: IfNotPresent
        name: manager
        env:
          - name: WATCH_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - "ALL"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
      serviceAccountName: primaza-agentapp
      terminationGracePeriodSeconds: 10
`
