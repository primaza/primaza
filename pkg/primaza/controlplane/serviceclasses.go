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

package controlplane

import (
	"context"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PushServiceClassToNamespaces(ctx context.Context, cli client.Client, sc primazaiov1alpha1.ServiceClass, namespaces []string) error {
	for _, ns := range namespaces {
		sccp := &primazaiov1alpha1.ServiceClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sc.Name,
				Namespace: ns,
			},
			Spec: sc.Spec,
		}

		if err := cli.Create(ctx, sccp, &client.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func DeleteServiceClassFromNamespaces(ctx context.Context, cli client.Client, sc primazaiov1alpha1.ServiceClass, namespaces []string) error {
	for _, ns := range namespaces {
		sccp := &primazaiov1alpha1.ServiceClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sc.Name,
				Namespace: ns,
			},
		}

		if err := cli.Delete(ctx, sccp, &client.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}
