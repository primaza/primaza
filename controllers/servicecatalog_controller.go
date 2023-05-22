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
	"errors"
	"fmt"

	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
	"github.com/primaza/primaza/pkg/primaza/controlplane"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

//+kubebuilder:rbac:groups=primaza.io.primaza.io,namespace=system,resources=servicecatalogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io.primaza.io,namespace=system,resources=servicecatalogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io.primaza.io,namespace=system,resources=servicecatalogs/finalizers,verbs=update

// ServiceCatalogReconciler reconciles a ServiceCatalog object
type ServiceCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ServiceCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconcile Service Catalog")

	// first, get the service catalog
	serviceCatalog := v1alpha1.ServiceCatalog{}
	err := r.Get(ctx, req.NamespacedName, &serviceCatalog)
	if err != nil {
		l.Error(err, "Failed to retrieve ServiceCatalog")
		if apierrors.IsNotFound(err) {
			// nothing to do as it means no ClusterEnvironment
			// exists for deleted ServiceCatalog
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// get a list of cluster environments
	clusterEnvironmentList := v1alpha1.ClusterEnvironmentList{}
	lo := client.ListOptions{Namespace: req.NamespacedName.Namespace}
	if err = r.List(ctx, &clusterEnvironmentList, &lo); err != nil {
		l.Error(err, "Error on listing clusterenvironment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var errorList []error
	for index := range clusterEnvironmentList.Items {
		ce := clusterEnvironmentList.Items[index]
		if ce.Spec.EnvironmentName == serviceCatalog.Name {
			if err := r.PushServiceCatalog(ctx, serviceCatalog, ce); err != nil {
				l.Error(err, fmt.Sprintf("ServiceCatalog:%v failed to be pushed to application namespaces of Cluster Envoronment:%v ", serviceCatalog, ce.Name))
				errorList = append(errorList, err)
			}
		}
	}
	return ctrl.Result{}, errors.Join(errorList...)
}

func (r *ServiceCatalogReconciler) PushServiceCatalog(ctx context.Context, serviceCatalog v1alpha1.ServiceCatalog, ce v1alpha1.ClusterEnvironment) error {
	l := log.FromContext(ctx)
	cfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
	if err != nil {
		return err
	}
	if err := controlplane.PushServiceCatalogToApplicationNamespaces(ctx, serviceCatalog, r.Scheme, r.Client, ce.Spec.ApplicationNamespaces, cfg); err != nil {
		l.Error(err, "error pushing service catalog")
		return err
	}
	return nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceCatalog{}).
		Complete(r)
}
