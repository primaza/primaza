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

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConnectionStatusReason string

const (
	ConnectionSuccessful ConnectionStatusReason = "ConnectionSuccessful"
	ConnectionError      ConnectionStatusReason = "ConnectionError"
	ClientCreationError  ConnectionStatusReason = "ClientCreationError"
)

type ConnectionStatus struct {
	State   primazaiov1alpha1.ClusterEnvironmentState
	Reason  ConnectionStatusReason
	Message string
}

func TestConnection(ctx context.Context, cfg *rest.Config) ConnectionStatus {
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return ConnectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  ClientCreationError,
			Message: fmt.Sprintf("error creating the client: %s", err),
		}
	}

	v, err := c.ServerVersion()
	if err != nil {
		return ConnectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  ConnectionError,
			Message: fmt.Sprintf("error connecting to target cluster: %s", err),
		}
	}

	return ConnectionStatus{
		State:   primazaiov1alpha1.ClusterEnvironmentStateOnline,
		Reason:  ConnectionSuccessful,
		Message: fmt.Sprintf("successfully connected to target cluster: kubernetes version found %s", v),
	}
}

func GetPrimazaKubeconfig(ctx context.Context, namespace string, cli client.Client, secretName string) (*rest.Config, string, error) {
	// TODO(sadlerap): can we use the functionality in GetClusterRESTConfig
	// from pkg/primaza/clustercontext to do de-duplicate this?
	s := v1.Secret{}
	k := client.ObjectKey{Namespace: namespace, Name: secretName}
	if err := cli.Get(ctx, k, &s); err != nil {
		return nil, "", err
	}
	if _, found := s.Data["kubeconfig"]; !found {
		return nil, "", fmt.Errorf("Field \"kubeconfig\" field in secret %s:%s does not exist", s.Name, s.Namespace)
	}

	if _, found := s.Data["namespace"]; !found {
		return nil, "", fmt.Errorf("Field \"namespace\" field in secret %s:%s does not exist", s.Name, s.Namespace)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(s.Data["kubeconfig"])
	if err != nil {
		return nil, "", err
	}
	return restConfig, string(s.Data["namespace"]), nil
}

func (c ConnectionStatus) Condition() metav1.Condition {
	status := func() metav1.ConditionStatus {
		if c.State == primazaiov1alpha1.ClusterEnvironmentStateOnline {
			return metav1.ConditionTrue
		}
		return metav1.ConditionFalse
	}()

	m := metav1.Condition{
		Type:    "Online",
		Reason:  string(c.Reason),
		Message: c.Message,
		Status:  status,
	}
	return m
}
