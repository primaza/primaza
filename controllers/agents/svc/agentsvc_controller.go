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

package svc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Agent Service Reconciler reconciles a Agent Service object
type AgentServiceReconciler struct {
	client.Client
}

func NewAgentServiceReconciler(mgr ctrl.Manager) *AgentServiceReconciler {
	return &AgentServiceReconciler{
		Client: mgr.GetClient(),
	}
}

const agentsvcfinalizer = "agent.primaza.io/finalizer"

func (r *AgentServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconcile Agent Service Deployment")

	// first, get the agent svc deployment
	agentsvcdeployment := appsv1.Deployment{}
	objKey := client.ObjectKey{
		Name:      constants.ServiceAgentDeploymentName,
		Namespace: req.Namespace,
	}
	err := r.Get(ctx, objKey, &agentsvcdeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// it should never happen that this controller does not find itself
			// FIXME: the deployment's been deleted, and the pod
			// we're running in is likely going to be deleted soon as well.  Do
			// we have a cleaner way of triggering our own shutdown?
			l.Error(err,
				"service agent deployment not found, that should be a bug",
				"expected deployment name", constants.ServiceAgentDeploymentName)
			os.Exit(1)
		}
		return ctrl.Result{}, err
	}
	// TODO: We need to ensure that we do not reconcile service classes once the agent service deployment
	// is marked for deletion.
	if agentsvcdeployment.DeletionTimestamp != nil {
		if err := r.removeServiceClasses(ctx, req); err != nil {
			return ctrl.Result{}, err
		}

		scc := v1alpha1.ServiceClassList{}
		if err := r.List(ctx, &scc, &client.ListOptions{Namespace: req.Namespace}); err != nil {
			return ctrl.Result{}, err
		}
		if len(scc.Items) > 0 {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("waiting for Service Classes deletion")
		}

		// remove the finalizer so we don't requeue
		if controllerutil.RemoveFinalizer(&agentsvcdeployment, agentsvcfinalizer) {
			if err = r.Update(ctx, &agentsvcdeployment, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else if !controllerutil.ContainsFinalizer(&agentsvcdeployment, agentsvcfinalizer) {
		// add a finalizer since we have deletion logic
		if controllerutil.AddFinalizer(&agentsvcdeployment, agentsvcfinalizer) {
			if err = r.Update(ctx, &agentsvcdeployment, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentServiceReconciler) removeServiceClasses(ctx context.Context, req ctrl.Request) error {
	// first, get the service class
	serviceclassesList := v1alpha1.ServiceClassList{}
	if err := r.List(ctx, &serviceclassesList, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		return client.IgnoreNotFound(err)
	}
	var errorList []error
	for _, scclass := range serviceclassesList.Items {
		serviceclass := scclass
		if err := r.Delete(ctx, &serviceclass, &client.DeleteOptions{}); err != nil {
			errorList = append(errorList, err)
		}
	}
	return errors.Join(errorList...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
