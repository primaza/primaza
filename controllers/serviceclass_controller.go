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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
	"github.com/primaza/primaza/pkg/primaza/controlplane"
)

// ServiceClassReconciler reconciles a ServiceClass object
type ServiceClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=primaza.io,resources=serviceclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,resources=serviceclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,resources=serviceclasses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceClass object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ServiceClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	sc := &primazaiov1alpha1.ServiceClass{}
	if err := r.Get(ctx, req.NamespacedName, sc, &client.GetOptions{}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// check if instance is marked to be deleted
	if sc.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(sc, clusterEnvironmentFinalizer) {
			// run finalizer
			if err := r.finalize(ctx, sc); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer from cluster environment
			controllerutil.RemoveFinalizer(sc, clusterEnvironmentFinalizer)
			if err := r.Update(ctx, sc); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// add finalizer if needed
	if !controllerutil.ContainsFinalizer(sc, clusterEnvironmentFinalizer) {
		controllerutil.AddFinalizer(sc, clusterEnvironmentFinalizer)
		if err := r.Update(ctx, sc); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileEnvironments(ctx, sc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceClassReconciler) finalize(ctx context.Context, sc *primazaiov1alpha1.ServiceClass) error {
	return r.removeFromEnvironments(ctx, sc)
}

func (r *ServiceClassReconciler) reconcileEnvironments(ctx context.Context, sc *primazaiov1alpha1.ServiceClass) error {
	cee := primazaiov1alpha1.ClusterEnvironmentList{}
	if err := r.List(ctx, &cee, &client.ListOptions{}); err != nil {
		return err
	}

	ff := r.filterClusterEnvironments(sc.Spec.Constraints.Environments, cee.Items)

	errs := []error{}
	for _, ce := range ff {
		cli, err := clustercontext.CreateClient(ctx, r.Client, ce, r.Scheme, r.Client.RESTMapper())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := controlplane.PushServiceClassToNamespaces(ctx, cli, *sc, ce.Spec.ServiceNamespaces); err != nil {
			errs = append(errs,
				fmt.Errorf("error pushing service class '%s' to cluster environment '%s': %w", sc.Name, ce.Name, err))
		}
	}

	return errors.Join(errs...)
}

func (r *ServiceClassReconciler) removeFromEnvironments(ctx context.Context, sc *primazaiov1alpha1.ServiceClass) error {
	ff, err := r.getRelatedClusterEnvironments(ctx, sc.Spec.Constraints.Environments)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, ce := range ff {
		cli, err := clustercontext.CreateClient(ctx, r.Client, ce, r.Scheme, r.Client.RESTMapper())
		if err != nil {
			return err
		}

		if err := controlplane.DeleteServiceClassFromNamespaces(ctx, cli, *sc, ce.Spec.ServiceNamespaces); err != nil {
			errs = append(errs,
				fmt.Errorf("error deleting service class '%s' to cluster environment '%s': %w", sc.Name, ce.Name, err))
		}
	}
	return errors.Join(errs...)
}

func (r *ServiceClassReconciler) getRelatedClusterEnvironments(ctx context.Context, constraints []string) ([]primazaiov1alpha1.ClusterEnvironment, error) {
	cee := primazaiov1alpha1.ClusterEnvironmentList{}
	if err := r.List(ctx, &cee, &client.ListOptions{}); err != nil {
		return nil, err
	}

	ff := r.filterClusterEnvironments(constraints, cee.Items)
	return ff, nil
}

func (r *ServiceClassReconciler) filterClusterEnvironments(
	environmentConstraints []string,
	clusterEnvironments []primazaiov1alpha1.ClusterEnvironment) []primazaiov1alpha1.ClusterEnvironment {

	cee := []primazaiov1alpha1.ClusterEnvironment{}
	for _, ce := range clusterEnvironments {
		if envtag.Match(ce.Spec.EnvironmentName, environmentConstraints) {
			cee = append(cee, ce)
		}
	}

	return cee
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceClass{}).
		Complete(r)
}
