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
	"errors"
	"fmt"

	"github.com/primaza/primaza/pkg/identity"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespacesBinder interface {
	BindNamespaces(ctx context.Context, ceName string, ceNamespace string, namespaces []string) error
}

func NewApplicationNamespacesBinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesBinder {
	return &namespacesBinder{pcli: primazaClient, wcli: workerClient, kind: ApplicationNamespaceType, pushAgent: workercluster.PushApplicationAgent}
}

func NewServiceNamespacesBinder(primazaClient client.Client, workerClient *kubernetes.Clientset) NamespacesBinder {
	return &namespacesBinder{pcli: primazaClient, wcli: workerClient, kind: ServiceNamespaceType, pushAgent: workercluster.PushServiceAgent}
}

type namespacesBinder struct {
	pcli client.Client
	wcli *kubernetes.Clientset
	kind NamespaceType

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
	i, err := b.createIdentity(ctx, ceName, ceNamespace, namespace)
	if err != nil {
		return err
	}

	if err := b.pushKubeconfigSecret(ctx, i, namespace); err != nil {
		return err
	}

	if err := b.pushAgent(ctx, b.wcli, namespace); err != nil {
		return err
	}

	return nil
}

func (b *namespacesBinder) pushKubeconfigSecret(ctx context.Context, i *identity.Instance, namespace string) error {
	kcfg, err := b.buildIdentityKubeconfig(ctx, i, namespace)
	if err != nil {
		return err
	}

	n := "kubeconfig-primaza-" + b.kind.Short()
	rs := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": kcfg,
			"namespace":  []byte(i.Namespace),
		},
	}
	if _, err := b.wcli.CoreV1().Secrets(namespace).Create(ctx, &rs, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}

		if _, err := b.wcli.CoreV1().Secrets(namespace).Update(ctx, &rs, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (b *namespacesBinder) buildIdentityKubeconfig(ctx context.Context, i *identity.Instance, namespace string) ([]byte, error) {
	if len(i.Secrets) == 0 {
		return nil, fmt.Errorf("no secret found for service account %s", i.ServiceAccount)
	}

	s := corev1.Secret{}
	if err := b.pcli.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: i.Secrets[0]}, &s); err != nil {
		return nil, err
	}

	t, err := identity.GetToken(&s)
	if err != nil {
		return nil, err
	}

	h, err := getExternalHost()
	if err != nil {
		return nil, err
	}

	return identity.GetKubeconfig(t, h, identity.GetKubeconfigOptions{
		User:      &i.ServiceAccount,
		Namespace: &i.Namespace,
	})
}

func (b *namespacesBinder) createIdentity(ctx context.Context, ceName, ceNamespace, namespace string) (*identity.Instance, error) {
	// get in cluster client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// create identity
	n := getAgentIdentityName(b.kind, ceName, namespace)
	i, err := identity.Create(ctx, *cli, n, ceNamespace)
	if err != nil {
		return nil, err
	}

	// create role bindings
	if err := b.createRoleBindings(ctx, ceName, ceNamespace, namespace, i.ServiceAccount); err != nil {
		return nil, err
	}

	return i, nil
}

func (b *namespacesBinder) createRoleBindings(ctx context.Context, ceName, ceNamespace, namespace, saname string) error {
	rr := getAgentRoleNames(b.kind)
	errs := []error{}
	for _, r := range rr {
		if err := b.createRoleBinding(ctx, ceName, ceNamespace, namespace, r, saname); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *namespacesBinder) createRoleBinding(ctx context.Context, ceName, ceNamespace, namespace, role, saname string) error {
	n := bakeRoleBindingName(role, ceName, namespace)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: ceNamespace,
			Labels: map[string]string{
				"app":                 "primaza",
				"tenant":              ceNamespace,
				"cluster-environment": ceName,
				"namespace-type":      string(b.kind),
				"namespace":           namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "User",
				Name:     fmt.Sprintf("primaza-%s-%s", ceName, namespace),
			},
			{
				APIGroup: "",
				Kind:     "ServiceAccount",
				Name:     saname,
			},
		},
	}
	if err := b.pcli.Create(ctx, rb, &client.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
