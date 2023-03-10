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

type NamespacesBinder interface {
	BindNamespaces(ctx context.Context, ceName string, ceNamespace string, namespaces []string) error
}

func NewApplicationNamespacesBinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesBinder {
	return &namespacesBinder{pcli: primazaClient, wcli: workerClient, kind: "application", pushAgent: workercluster.PushApplicationAgent}
}

func NewServiceNamespacesBinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesBinder {
	return &namespacesBinder{pcli: primazaClient, wcli: workerClient, kind: "service", pushAgent: workercluster.PushServiceAgent}
}

type namespacesBinder struct {
	pcli client.Client
	wcli *kubernetes.Clientset
	kind string

	pushAgent func(context.Context, *kubernetes.Clientset, string) error
}

func (b *namespacesBinder) BindNamespaces(ctx context.Context, ceName string, ceNamespace string, namespaces []string) error {
	l := log.FromContext(ctx)

	ens := []string{}
	for _, ns := range namespaces {
		if err := b.bindNamespace(ctx, ceName, ceNamespace, ns); err != nil {
			ens = append(ens, ns)
			l.Error(err, "error binding namespace", "cluster-environment", ceName, "namespace", ns)
		}
	}

	if len(ens) != 0 {
		return fmt.Errorf("error binding namespaces: %v", ens)
	}
	return nil
}

func (b *namespacesBinder) bindNamespace(ctx context.Context, ceName, ceNamespace string, namespace string) error {
	if err := b.createRoleBinding(ctx, ceName, ceNamespace, namespace); err != nil {
		return err
	}

	if err := b.pushAgent(ctx, b.wcli, namespace); err != nil {
		return err
	}

	return nil
}

func (b *namespacesBinder) createRoleBinding(ctx context.Context, ceName, ceNamespace, namespace string) error {
	n := b.getRoleBindingName(ceName, namespace)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: ceNamespace,
			Labels: map[string]string{
				"app":                 "primaza",
				"cluster-environment": ceName,
				"namespace-type":      b.kind,
				"namespace":           namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     fmt.Sprintf("primaza-%s-agent-role", b.kind),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "User",
				Name:     n,
			},
		},
	}

	if err := b.pcli.Create(ctx, rb, &client.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (b *namespacesBinder) getRoleBindingName(ceName, namespace string) string {
	return fmt.Sprintf("primaza-%s-%s", ceName, namespace)
}
