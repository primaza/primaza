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
	"fmt"

	"github.com/primaza/primaza/pkg/primaza/workercluster"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespacesUnbinder interface {
	UnbindNamespaces(context.Context, string, string, []string) error
}

func NewApplicationNamespacesUnbinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesUnbinder {
	return &namespacesUnbinder{
		pcli:        primazaClient,
		wcli:        workerClient,
		kind:        ApplicationNamespaceType,
		deleteAgent: workercluster.DeleteApplicationAgent,
	}
}

func NewServiceNamespacesUnbinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesUnbinder {
	return &namespacesUnbinder{
		pcli:        primazaClient,
		wcli:        workerClient,
		kind:        ServiceNamespaceType,
		deleteAgent: workercluster.DeleteServiceAgent,
	}
}

type namespacesUnbinder struct {
	pcli client.Client
	wcli *kubernetes.Clientset
	kind NamespaceType

	deleteAgent func(context.Context, *kubernetes.Clientset, string) error
}

func (b *namespacesUnbinder) UnbindNamespaces(ctx context.Context, ceName, ceNamespace string, namespaces []string) error {
	l := log.FromContext(ctx)

	ens := []string{}
	for _, ns := range namespaces {
		if err := b.unbindNamespace(ctx, ceName, ceNamespace, ns); err != nil {
			ens = append(ens, ns)
			l.Error(err, "error unbinding namespace", "cluster-environment", ceName, "namespace", ns)
		}
	}

	if len(ens) != 0 {
		return fmt.Errorf("error unbinding namespaces: %v", ens)
	}

	return nil
}

func (b *namespacesUnbinder) unbindNamespace(ctx context.Context, ceName, ceNamespace string, namespace string) error {
	if err := b.deleteRoleBinding(ctx, ceName, ceNamespace, namespace); err != nil && !errors.IsNotFound(err) {
		return err
	}

	if err := b.deleteAgent(ctx, b.wcli, namespace); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func (b *namespacesUnbinder) deleteRoleBinding(ctx context.Context, ceName, ceNamespace, namespace string) error {
	n := b.getRoleBindingName(ceName, namespace)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: ceNamespace,
		},
	}
	if err := b.pcli.Delete(ctx, rb, &client.DeleteOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (b *namespacesUnbinder) getRoleBindingName(ceName, namespace string) string {
	return fmt.Sprintf("primaza-%s-%s", ceName, namespace)
}
