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
	"os"

	"github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Agent Application Reconciler reconciles a Agent Application object
type AgentApplicationReconciler struct {
	client.Client
}

func NewAgentApplicationReconciler(mgr ctrl.Manager) *AgentApplicationReconciler {
	return &AgentApplicationReconciler{
		Client: mgr.GetClient(),
	}
}

const agentappfinalizer = "agentapp.primaza.io/finalizer"

func (r *AgentApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconcile Agent app Deployment")

	// first, get the agent app deployment
	agentappdeployment := appsv1.Deployment{}
	objKey := client.ObjectKey{
		Name:      constants.ApplicationAgentDeploymentName,
		Namespace: req.Namespace,
	}
	err := r.Get(ctx, objKey, &agentappdeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// it should never happen that this controller does not find itself
			// FIXME: the deployment's been deleted, and the pod
			// we're running in is likely going to be deleted soon as well.  Do
			// we have a cleaner way of triggering our own shutdown?
			l.Error(err,
				"application agent deployment not found, that should be a bug",
				"expected deployment name", constants.ApplicationAgentDeploymentName)
			os.Exit(1)
		}
		return ctrl.Result{}, err
	}
	// TODO: We need to ensure that we do not reconcile service catalog and service binding once the agent application deployment
	// is marked for deletion.
	if !agentappdeployment.DeletionTimestamp.IsZero() {
		if err := r.removePrimazaResources(ctx, req); err != nil {
			l.Error(err, "Service catalog and service binding deletion failed")
			return ctrl.Result{}, err
		}
		// remove the finalizer so we don't requeue
		if controllerutil.RemoveFinalizer(&agentappdeployment, agentappfinalizer) {
			if err = r.Update(ctx, &agentappdeployment, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else if !controllerutil.ContainsFinalizer(&agentappdeployment, agentappfinalizer) {
		// add a finalizer since we have deletion logic
		if controllerutil.AddFinalizer(&agentappdeployment, agentappfinalizer) {
			if err = r.Update(ctx, &agentappdeployment, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentApplicationReconciler) removePrimazaResources(ctx context.Context, req ctrl.Request) error {
	errs := []error{}
	if err := r.removeServiceCatalog(ctx, req); err != nil {
		errs = append(errs, err)
	}
	if err := r.removeServiceBinding(ctx, req); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (r *AgentApplicationReconciler) removeServiceCatalog(ctx context.Context, req ctrl.Request) error {
	return client.IgnoreNotFound(
		r.DeleteAllOf(ctx,
			&v1alpha1.ServiceCatalog{},
			client.InNamespace(req.Namespace)))
}

func (r *AgentApplicationReconciler) removeServiceBinding(ctx context.Context, req ctrl.Request) error {
	return client.IgnoreNotFound(
		r.DeleteAllOf(ctx,
			&v1alpha1.ServiceBinding{},
			client.InNamespace(req.Namespace)))
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	filter := func(c client.Object) bool {
		// we're only interested in watching the application agent, so filter
		// out every deployment in our namespace besides our own deployment
		return c.GetName() == constants.ApplicationAgentDeploymentName
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.NewPredicateFuncs(filter)).
		For(&appsv1.Deployment{}).
		Complete(r)
}
