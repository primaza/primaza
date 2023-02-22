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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ConnectionStatus struct {
	State   primazaiov1alpha1.ClusterEnvironmentState
	Reason  string
	Message string
}

func TestConnection(ctx context.Context, cfg *rest.Config) ConnectionStatus {
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return ConnectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  "ClientCreationError",
			Message: fmt.Sprintf("error creating the client: %s", err),
		}
	}

	v, err := c.ServerVersion()
	if err != nil {
		return ConnectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  "ConnectionError",
			Message: fmt.Sprintf("error connecting to target cluster: %s", err),
		}
	}

	return ConnectionStatus{
		State:   primazaiov1alpha1.ClusterEnvironmentStateOnline,
		Reason:  "ConnectionSuccessful",
		Message: fmt.Sprintf("successfully connected to target cluster: kubernetes version found %s", v),
	}
}
