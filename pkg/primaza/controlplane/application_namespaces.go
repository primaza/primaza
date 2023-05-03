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

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func PushServiceBinding(
	ctx context.Context,
	sc *primazaiov1alpha1.ServiceClaim,
	secret *corev1.Secret,
	scheme *runtime.Scheme,
	controllerruntimeClient client.Client,
	nspace *string,
	applicationNamespaces []string,
	cfg *rest.Config) error {
	l := log.FromContext(ctx)
	oc := client.Options{
		Scheme: scheme,
		Mapper: controllerruntimeClient.RESTMapper(),
	}
	cecli, err := client.New(cfg, oc)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, ns := range applicationNamespaces {
		if nspace == nil || *nspace == ns {
			l.Info("pushing to application namespace", "application namespace", ns)
			if err := pushServiceBindingToNamespace(ctx, cecli, ns, sc, secret); err != nil {
				errs = append(errs, err)
				l.Error(err, "error pushing to application namespaces", "application namespace", ns)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func pushServiceBindingToNamespace(
	ctx context.Context,
	cli client.Client,
	namespace string,
	sc *primazaiov1alpha1.ServiceClaim,
	secret *corev1.Secret) error {
	l := log.FromContext(ctx)

	sb := primazaiov1alpha1.ServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sc.Name,
			Namespace: namespace,
		},
		Spec: primazaiov1alpha1.ServiceBindingSpec{
			ServiceEndpointDefinitionSecret: sc.Name,
			Application:                     sc.Spec.Application,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, cli, &sb, func() error {
		sb.Spec = primazaiov1alpha1.ServiceBindingSpec{
			ServiceEndpointDefinitionSecret: sc.Name,
			Application:                     sc.Spec.Application,
		}
		return nil
	})

	if err != nil {
		l.Error(err, "Failed to create or update service binding")
		return err
	} else {
		l.Info("Wrote service binding", "binding", sb.Name, "namespace", sb.Namespace, "operation", op)
	}

	if err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: sc.Name}, &sb); err != nil {
		return err
	}

	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "primaza.io/v1alpha1",
			Kind:       "ServiceBinding",
			Name:       sc.Name,
			UID:        sb.UID,
		},
	}
	secret.Namespace = namespace
	data := secret.StringData
	l.Info("creating or updating secret for service claim", "secret", secret, "service claim", sc)
	op, err = controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
		secret.StringData = data
		return nil
	})

	if err != nil {
		l.Error(err, "error creating or updating secret for service claim", "secret", secret, "service claim", sc)
		return err
	} else {
		l.Info("Wrote secret", "secret", secret.Name, "namespace", secret.Namespace, "operation", op)
	}

	return nil
}

func PushServiceCatalogToApplicationNamespaces(ctx context.Context, sc primazaiov1alpha1.ServiceCatalog, scheme *runtime.Scheme, controllerruntimeClient client.Client, applicationNamespaces []string, cfg *rest.Config) error {
	l := log.FromContext(ctx)
	oc := client.Options{
		Scheme: scheme,
		Mapper: controllerruntimeClient.RESTMapper(),
	}
	cli, err := client.New(cfg, oc)
	if err != nil {
		return err
	}
	var errorList []error
	for _, ns := range applicationNamespaces {
		sccp := &primazaiov1alpha1.ServiceCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sc.Name,
				Namespace: ns,
			},
		}

		op, err := controllerutil.CreateOrUpdate(ctx, cli, sccp, func() error {
			sccp.Spec = sc.Spec
			return nil
		})

		if err != nil {
			l.Error(err, "Failed to create or update service catalog")
			errorList = append(errorList, err)
		} else {
			l.Info("Wrote service catalog", "catalog", sccp.Name, "namespace", sccp.Namespace, "operation", op)
		}
	}

	return errors.Join(errorList...)
}

func DeleteServiceBindingAndSecretFromNamespaces(ctx context.Context, cli client.Client, sc primazaiov1alpha1.ServiceClaim, namespaces []string) error {
	var errs []error

	for _, ns := range namespaces {
		sb := &primazaiov1alpha1.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sc.Name,
				Namespace: ns,
			},
		}

		if err := cli.Delete(ctx, sb, &client.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, err)
			}

		}
	}

	return errors.Join(errs...)
}
