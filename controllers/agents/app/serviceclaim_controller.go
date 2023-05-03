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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	sccontrollers "github.com/primaza/primaza/controllers"
	"github.com/primaza/primaza/pkg/primaza/constants"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
)

const ServiceClaimFinalizer = "serviceclaims.primaza.io/finalizer"

// ServiceClaimReconciler reconciles a ServiceClaim object
type ServiceClaimReconciler struct {
	sccontrollers.ServiceClaimReconciler
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceClaim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ServiceClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	l.Info("Reconciling service claim")

	var sclaim primazaiov1alpha1.ServiceClaim
	if err := r.Get(ctx, req.NamespacedName, &sclaim); err != nil {
		l.Error(err, "Failed to retrieve ServiceClaim")
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	config, remote_namespace, err := workercluster.GetPrimazaKubeconfig(ctx, sclaim.Namespace, r.Client, constants.ApplicationAgentKubeconfigSecretName)
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("remote cluster", "address", config.Host)

	remote_client, err := client.New(config, client.Options{
		Scheme: r.Client.Scheme(),
		Mapper: r.Mapper,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	objKey := client.ObjectKey{
		Name:      constants.ApplicationAgentDeploymentName,
		Namespace: req.NamespacedName.Namespace,
	}
	var deployment appsv1.Deployment
	err = r.Get(ctx, objKey, &deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// it should never happen that this controller does not find itself
			l.Error(err,
				"application agent deployment not found, that should be a bug",
				"expected deployment name", constants.ApplicationAgentDeploymentName)
			panic(err)
		}
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if sclaim.DeletionTimestamp.IsZero() && deployment.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&sclaim, ServiceClaimFinalizer) {
			controllerutil.AddFinalizer(&sclaim, ServiceClaimFinalizer)
			if err := r.Update(ctx, &sclaim); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&sclaim, ServiceClaimFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			sclaimCopy := r.createServiceClaimCopy(sclaim, deployment, remote_namespace)
			if err := r.deleteExternalResources(ctx, sclaimCopy, remote_client); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&sclaim, ServiceClaimFinalizer)
			if err := r.Update(ctx, &sclaim); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	sclaimCopy := r.createServiceClaimCopy(sclaim, deployment, remote_namespace)
	spec := sclaimCopy.Spec
	op, err := controllerutil.CreateOrUpdate(ctx, remote_client, sclaimCopy, func() error {
		sclaimCopy.Spec = spec
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "admission webhook \"vserviceclaim.kb.io\" denied the request") {
			sclaimCopy.Status.State = primazaiov1alpha1.ServiceClaimStateInvalid
		} else {
			sclaimCopy.Status.State = primazaiov1alpha1.ServiceClaimStatePending
			l.Error(err, "Failed to create/update service claim",
				"service", sclaim.Name,
				"namespace", sclaim.Namespace)
		}

	} else {
		sclaimCopy.Status.State = primazaiov1alpha1.ServiceClaimStateResolved
		l.Info("Wrote service claim", "claim", sclaim.Name, "namespace", sclaim.Namespace, "operation", op)
	}
	sclaim.Status.ClaimID = sclaimCopy.Status.ClaimID
	sclaim.Status.State = sclaimCopy.Status.State
	sclaim.Status.RegisteredService = sclaimCopy.Status.RegisteredService
	if err := r.Status().Update(ctx, &sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceClaimReconciler) createServiceClaimCopy(sclaim primazaiov1alpha1.ServiceClaim, deployment appsv1.Deployment, remote_namespace string) *primazaiov1alpha1.ServiceClaim {
	sclaimCopy := sclaim.DeepCopy()
	sclaimCopy.Spec.EnvironmentTag = ""
	sclaimCopy.Spec.ApplicationClusterContext = &primazaiov1alpha1.ServiceClaimApplicationClusterContext{}
	sclaimCopy.Spec.ApplicationClusterContext.ClusterEnvironmentName = deployment.Labels["primaza.io/cluster-environment"]
	sclaimCopy.Spec.ApplicationClusterContext.Namespace = sclaim.Namespace
	sclaimCopy.Namespace = remote_namespace
	sclaimCopy.ResourceVersion = ""
	return sclaimCopy
}

func (r *ServiceClaimReconciler) deleteExternalResources(ctx context.Context, sclaim *primazaiov1alpha1.ServiceClaim, cli client.Client) error {
	if err := cli.Delete(ctx, sclaim); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceClaim{}).
		Complete(r)
}
