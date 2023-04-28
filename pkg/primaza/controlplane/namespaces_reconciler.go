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

	"github.com/primaza/primaza/pkg/slices"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ClusterEnvironmentState struct {
	Name          string
	Namespace     string
	ClusterConfig *rest.Config

	ApplicationNamespaces []string
	ServiceNamespaces     []string
}

type NamespacesReconciler interface {
	ReconcileNamespaces(ctx context.Context) error
}

type namespacesReconciler struct {
	pcli client.Client
	env  ClusterEnvironmentState

	appBinder NamespacesBinder
	svcBinder NamespacesBinder

	appUnbinder NamespacesUnbinder
	svcUnbinder NamespacesUnbinder
}

func NewNamespaceReconciler(e ClusterEnvironmentState) (NamespacesReconciler, error) {
	cli, err := getInClusterClient()
	if err != nil {
		return nil, err
	}

	wcli, err := kubernetes.NewForConfig(e.ClusterConfig)
	if err != nil {
		return nil, err
	}

	return &namespacesReconciler{
		pcli:        cli,
		env:         e,
		appBinder:   NewApplicationNamespacesBinder(cli, wcli),
		appUnbinder: NewApplicationNamespacesUnbinder(cli, wcli),
		svcBinder:   NewServiceNamespacesBinder(cli, wcli),
		svcUnbinder: NewServiceNamespacesUnbinder(cli, wcli),
	}, nil
}

func (r *namespacesReconciler) ReconcileNamespaces(ctx context.Context) error {
	errs := []error{}

	if err := r.bindNamespaces(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := r.unbindOrphanNamespaces(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func (r *namespacesReconciler) bindNamespaces(ctx context.Context) error {
	aerr := r.appBinder.BindNamespaces(ctx, r.env.Name, r.env.Namespace, r.env.ApplicationNamespaces)
	serr := r.svcBinder.BindNamespaces(ctx, r.env.Name, r.env.Namespace, r.env.ServiceNamespaces)

	return errors.Join(aerr, serr)
}

func (r *namespacesReconciler) unbindOrphanNamespaces(ctx context.Context) error {
	aerr := r.unbindOrphanNamespacesForType(ctx, r.appUnbinder, ApplicationNamespaceType, r.env.ApplicationNamespaces)
	serr := r.unbindOrphanNamespacesForType(ctx, r.svcUnbinder, ServiceNamespaceType, r.env.ServiceNamespaces)

	return errors.Join(aerr, serr)
}

func (r *namespacesReconciler) unbindOrphanNamespacesForType(ctx context.Context, ub NamespacesUnbinder, namespaceType NamespaceType, namespaces []string) error {
	nn, err := r.getOrphanNamespaces(ctx, r.env.Name, namespaceType, namespaces)
	if err != nil {
		return err
	}
	l := log.FromContext(ctx)
	l.Info("unbinding orphan namespaces", "namespace-type", namespaceType, "orphan-namespaces", nn)

	return ub.UnbindNamespaces(ctx, r.env.Name, r.env.Namespace, nn)
}

func (r *namespacesReconciler) getOrphanNamespaces(ctx context.Context, ceName string, namespaceType NamespaceType, requestedNamespaces []string) ([]string, error) {
	ann, err := r.getAuthorizedNamespaces(ctx, r.env.Name, namespaceType)
	if err != nil {
		return nil, err
	}

	return slices.SubtractStr(ann, requestedNamespaces), nil
}

func (r *namespacesReconciler) getAuthorizedNamespaces(ctx context.Context, ceName string, namespaceType NamespaceType) ([]string, error) {
	l := log.FromContext(ctx)

	rbb := &rbacv1.RoleBindingList{}
	ls := getRoleBindingsLabelSelectorOrDie(ceName, namespaceType)
	if err := r.pcli.List(ctx, rbb, &client.ListOptions{LabelSelector: ls, Namespace: r.env.Namespace}); err != nil {
		return nil, err
	}

	nss := []string{}
	for _, rb := range rbb.Items {
		ns, ok := rb.GetLabels()["primaza.io/namespace"]
		if !ok {
			l.Info("can't find namespace label in Primaza's agent Role Binding", "role-binding", rb)
			continue
		}
		nss = append(nss, ns)
	}
	return nss, nil
}
