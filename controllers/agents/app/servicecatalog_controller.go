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

package controllers

import (
	"context"

	"github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	app1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceCatalogReconciler reconciles a ServiceCatalog object
type ServiceCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceCatalog object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ServiceCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcileLog := log.FromContext(ctx)
	reconcileLog.Info("Reconciling service catalog in agent app", "namespace", req.Namespace, "name", req.Name)

	serviceCatalog := v1alpha1.ServiceCatalog{}
	err := r.Get(ctx, req.NamespacedName, &serviceCatalog)

	if err != nil {
		reconcileLog.Error(err, "Failed to retrieve ServiceCatalog", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}
	if err = r.setOwnerReference(ctx, &serviceCatalog, req.Namespace); err != nil {
		reconcileLog.Error(err, "Failed to set owner reference on ServiceCatalog", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ServiceCatalogReconciler) setOwnerReference(ctx context.Context, scat *v1alpha1.ServiceCatalog, namespace string) error {
	reconcileLog := log.FromContext(ctx)
	objKey := client.ObjectKey{
		Name:      constants.ApplicationAgentDeploymentName,
		Namespace: namespace,
	}
	var deployment app1.Deployment
	if err := r.Get(ctx, objKey, &deployment); err != nil {
		reconcileLog.Error(err, "unable to retrieve agent app deployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests
		return client.IgnoreNotFound(err)
	}
	if len(deployment.OwnerReferences) == 0 {
		if err := ctrl.SetControllerReference(&deployment, scat, r.Scheme); err != nil {
			return err
		}
	} else {
		var found bool
		for _, owner := range deployment.OwnerReferences {
			if owner.Kind == "ServiceCatalog" {
				found = true
				break
			}
		}
		if !found {
			if err := ctrl.SetControllerReference(&deployment, scat, r.Scheme); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceCatalog{}).
		Complete(r)
}
