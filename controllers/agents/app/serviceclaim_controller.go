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
	"fmt"
	"os"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
)

const ServiceClaimFinalizer = "serviceclaims.primaza.io/finalizer"

// ServiceClaimReconciler reconciles a ServiceClaim object
type ServiceClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Mapper meta.RESTMapper
}

func NewServiceClaimReconciler(mgr ctrl.Manager) *ServiceClaimReconciler {
	return &ServiceClaimReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	}
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

	config, remote_namespace, err := workercluster.GetPrimazaKubeconfig(ctx)
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

	{ // runtime service claim validation
		ok, err := r.ensureServiceClaimIsValid(ctx, &sclaim)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !ok {
			return ctrl.Result{}, nil
		}
	}

	sclaimCopy := r.createServiceClaimCopy(sclaim, deployment, remote_namespace)
	spec := sclaimCopy.Spec

	l.Info("Wrote service claim", "claim", sclaim.Name, "namespace", sclaim.Namespace)
	if _, err := controllerutil.CreateOrUpdate(ctx, remote_client, sclaimCopy, func() error {
		sclaimCopy.Spec = spec
		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "admission webhook \"vserviceclaim.kb.io\" denied the request") {
			c := metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               string(primazaiov1alpha1.ServiceClaimConditionReady),
				Status:             metav1.ConditionFalse,
				Reason:             constants.ValidationErrorReason,
				Message:            err.Error(),
			}
			meta.SetStatusCondition(&sclaim.Status.Conditions, c)
			sclaim.Status.State = primazaiov1alpha1.ServiceClaimStateInvalid

			if err := r.Status().Update(ctx, &sclaim); err != nil {
				l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
				return ctrl.Result{}, err
			}

		} else {
			l.Error(err, "Failed to create/update service claim",
				"service", sclaim.Name,
				"namespace", sclaim.Namespace)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceClaimReconciler) ensureServiceClaimIsValid(
	ctx context.Context,
	sclaim *primazaiov1alpha1.ServiceClaim,
) (bool, error) {
	// check immutable properties
	pp := r.getUpdatedImmutableProperties(ctx, sclaim)
	if len(pp) > 0 {
		c := metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.TypeValidResource,
			Status:             metav1.ConditionFalse,
			Reason:             primazaiov1alpha1.ReasonUpdatedImmutableField,
			Message:            fmt.Sprintf(`%s can not be updated`, strings.Join(pp, ", ")),
		}
		return false, r.setStatusConditionAndInvalidStateIfApplicable(ctx, sclaim, c)
	}

	// check spec.target
	if sclaim.Spec.Target != nil {
		c := metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.TypeValidResource,
			Status:             metav1.ConditionFalse,
			Reason:             primazaiov1alpha1.ReasonForbiddenField,
			Message:            `".spec.target" can not be defined in application namespace`,
		}
		return false, r.setStatusConditionAndInvalidStateIfApplicable(ctx, sclaim, c)
	}

	return true, nil
}

func (r *ServiceClaimReconciler) setStatusConditionAndInvalidStateIfApplicable(
	ctx context.Context,
	sclaim *primazaiov1alpha1.ServiceClaim,
	condition metav1.Condition,
) error {
	meta.SetStatusCondition(&sclaim.Status.Conditions, condition)
	if sclaim.Status.State == "" {
		sclaim.Status.State = primazaiov1alpha1.ServiceClaimStateInvalid
	}
	return r.Client.Status().Update(ctx, sclaim)
}

func (r *ServiceClaimReconciler) getUpdatedImmutableProperties(
	ctx context.Context,
	sclaim *primazaiov1alpha1.ServiceClaim,
) []string {
	pp := []string{}
	if sclaim.Status.OriginalServiceClassIdentity != nil &&
		!reflect.DeepEqual(sclaim.Status.OriginalServiceClassIdentity, sclaim.Spec.ServiceClassIdentity) {
		pp = append(pp, ".spec.serviceClassIdentity")
	}

	if sclaim.Status.OriginalServiceEndpointDefinitionKeys != nil &&
		!reflect.DeepEqual(sclaim.Status.OriginalServiceEndpointDefinitionKeys, sclaim.Spec.ServiceEndpointDefinitionKeys) {
		pp = append(pp, ".spec.ServiceEndpointDefinitionKeys")
	}
	return pp
}

func (r *ServiceClaimReconciler) createServiceClaimCopy(sclaim primazaiov1alpha1.ServiceClaim, deployment appsv1.Deployment, remote_namespace string) *primazaiov1alpha1.ServiceClaim {
	sclaimCopy := sclaim.DeepCopy()
	sclaimCopy.Spec.Target = &primazaiov1alpha1.ServiceClaimTarget{
		EnvironmentTag: "",
		ApplicationClusterContext: &primazaiov1alpha1.ServiceClaimApplicationClusterContext{
			ClusterEnvironmentName: deployment.Labels["primaza.io/cluster-environment"],
			Namespace:              sclaim.Namespace,
		},
	}
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
