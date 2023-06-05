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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/uuid"
	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
	"github.com/primaza/primaza/pkg/primaza/constants"
	"github.com/primaza/primaza/pkg/primaza/controlplane"
	"github.com/primaza/primaza/pkg/slices"
)

// ServiceClaimReconciler reconciles a ServiceClaim object
type ServiceClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Mapper meta.RESTMapper
}

const ServiceClaimFinalizer = "serviceclaims.primaza.io/finalizer"

func NewServiceClaimReconciler(mgr ctrl.Manager) *ServiceClaimReconciler {
	return &ServiceClaimReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	}
}

//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=serviceclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=serviceclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=serviceclaims/finalizers,verbs=update

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
	l := log.FromContext(ctx)

	l.Info("starting  service claim reconciliation")
	defer l.Info("reconciliation ended")

	var sclaim primazaiov1alpha1.ServiceClaim
	if err := r.Get(ctx, req.NamespacedName, &sclaim); err != nil {
		l.Info("unable to retrieve ServiceClaim", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Check if Service Claim is marked for deletion")
	if sclaim.HasDeletionTimestamp() {
		if controllerutil.ContainsFinalizer(&sclaim, ServiceClaimFinalizer) {
			if err := r.processClaimMarkedForDeletion(ctx, req, sclaim); err != nil {
				return ctrl.Result{}, err
			}
			// Remove finalizer from service binding
			if finalizerBool := controllerutil.RemoveFinalizer(&sclaim, ServiceClaimFinalizer); !finalizerBool {
				l.Error(errors.New("Finalizer not removed for service claim"), "Finalizer not removed for service claim")
				return ctrl.Result{}, errors.New("Finalizer not removed for service claim")
			}
			if err := r.Update(ctx, &sclaim); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	l.Info("Add Finalizer if needed")
	// add finalizer if needed
	if !controllerutil.ContainsFinalizer(&sclaim, ServiceClaimFinalizer) {
		controllerutil.AddFinalizer(&sclaim, ServiceClaimFinalizer)
		if err := r.Update(ctx, &sclaim); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch sclaim.Status.State {
	case "":
		sclaim.Status.ClaimID = uuid.New().String()
		l.Info("reconciling new service claim")
		return ctrl.Result{}, r.processPendingClaim(ctx, req, sclaim)
	case primazaiov1alpha1.ServiceClaimStatePending:
		l.Info("reconciling pending service claim")
		return ctrl.Result{}, r.processPendingClaim(ctx, req, sclaim)
	default:
		l.Info("reconciling resolved service claim")
		return ctrl.Result{}, nil
	}
}

func (r *ServiceClaimReconciler) processPendingClaim(ctx context.Context, req ctrl.Request, sclaim primazaiov1alpha1.ServiceClaim) error {
	l := log.FromContext(ctx)

	var rsl primazaiov1alpha1.RegisteredServiceList
	lo := client.ListOptions{Namespace: req.NamespacedName.Namespace}
	if err := r.List(ctx, &rsl, &lo); err != nil {
		l.Info("unable to retrieve RegisteredServiceList", "error", err)
		return client.IgnoreNotFound(err)
	}

	err := r.processServiceClaim(ctx, req, rsl, sclaim)
	if err != nil {
		l.Error(err, "unable while processing ServiceClaim")
		return err
	}
	return nil
}

func (r *ServiceClaimReconciler) processClaimMarkedForDeletion(ctx context.Context, req ctrl.Request, sclaim primazaiov1alpha1.ServiceClaim) error {
	l := log.FromContext(ctx)
	errs := []error{}

	var rsl primazaiov1alpha1.RegisteredServiceList
	lo := client.ListOptions{Namespace: req.NamespacedName.Namespace}
	if err := r.List(ctx, &rsl, &lo); err != nil {
		l.Info("unable to retrieve RegisteredServiceList", "error", err)
		errs = append(errs, client.IgnoreNotFound(err))
	}
	var registeredServiceFound bool
	var registeredService primazaiov1alpha1.RegisteredService
	for _, rs := range rsl.Items {
		// Check if the ServiceClassIdentity given in ServiceClaim is a subset of
		// ServiceClassIdentity given in the RegisteredService
		if checkSCISubset(sclaim.Spec.ServiceClassIdentity, rs.Spec.ServiceClassIdentity) &&
			(rs.Spec.Constraints == nil ||
				envtag.Match(sclaim.Spec.EnvironmentTag, rs.Spec.Constraints.Environments)) {
			registeredServiceFound = true
			registeredService = rs
			break
		}
	}

	if err := r.DeleteServiceBindingsAndSecret(ctx, req, sclaim); err != nil {
		l.Error(err, "unable to delete service binding and secret", "Service Binding", sclaim.Name)
		errs = append(errs, err)
	}

	if registeredServiceFound {
		if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateAvailable); err != nil {
			l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Ref. https://stackoverflow.com/a/18879994/547840
func checkSCISubset(serviceClaim, registeredService []v1alpha1.ServiceClassIdentityItem) bool {
	set := make(map[v1alpha1.ServiceClassIdentityItem]int)
	for _, value := range registeredService {
		set[value] += 1
	}

	for _, value := range serviceClaim {
		if count, found := set[value]; !found {
			return false
		} else if count < 1 {
			return false
		} else {
			set[value] = count - 1
		}
	}

	return true
}

func (r *ServiceClaimReconciler) extractServiceEndpointDefinition(
	ctx context.Context,
	req ctrl.Request,
	rs v1alpha1.RegisteredService,
	sedKeys []string,
	secret *corev1.Secret) (int, error) {
	l := log.FromContext(ctx)
	count := 0

	// loop over the ServiceEndpointDefinition array part of RegisteredService
	for _, sed := range rs.Spec.ServiceEndpointDefinition {
		// check if the value is non-empty
		if sed.Value != "" {
			// check if the ServiceEndpointDefinitionKeys part of ServiceClaim has the current
			// SED name in the RegisteredService
			if slices.ItemContains(sedKeys, sed.Name) {
				secret.StringData[sed.Name] = sed.Value
				count++
			}
		} else if k := sed.ValueFromSecret.Key; k != "" { // check value if the key is non-empty
			if slices.ItemContains(sedKeys, sed.Name) {
				sec := &corev1.Secret{}
				nn := types.NamespacedName{Namespace: req.Namespace, Name: sed.ValueFromSecret.Name}
				if err := r.Get(ctx, nn, sec); err != nil {
					l.Info("unable to retrieve Secret", "error", err, "secret", nn)
					continue
				}

				secret.StringData[sed.Name] = string(sec.Data[k])
				count++
			}
		}
	}
	return count, nil
}

func (r *ServiceClaimReconciler) changeServiceState(ctx context.Context, rs primazaiov1alpha1.RegisteredService, state string) error {
	rs.Status.State = state
	if err := r.Status().Update(ctx, &rs); err != nil {
		return err
	}

	return nil
}

func (r *ServiceClaimReconciler) getEnvironmentFromClusterEnvironment(
	ctx context.Context,
	req ctrl.Request,
	clusterEnvironmentName string) (*primazaiov1alpha1.ClusterEnvironment, error) {
	l := log.FromContext(ctx)
	ce := &primazaiov1alpha1.ClusterEnvironment{}
	objectKey := types.NamespacedName{Name: clusterEnvironmentName, Namespace: req.NamespacedName.Namespace}
	if err := r.Get(ctx, objectKey, ce); err != nil {
		l.Info("unable to retrieve ClusterEnvironment", "error", err)
		return nil, client.IgnoreNotFound(err)
	}

	return ce, nil
}

func (r *ServiceClaimReconciler) processServiceClaim(
	ctx context.Context,
	req ctrl.Request,
	rsl primazaiov1alpha1.RegisteredServiceList,
	sclaim primazaiov1alpha1.ServiceClaim) error {
	l := log.FromContext(ctx)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.NamespacedName.Name,
			Namespace: req.NamespacedName.Namespace,
		},
		StringData: map[string]string{},
	}

	// count the number of secret data entries
	count := 0
	// loop over every RegisteredService
	registeredServiceFound := false
	var registeredService primazaiov1alpha1.RegisteredService
	env := sclaim.Spec.EnvironmentTag
	if sclaim.Spec.ApplicationClusterContext != nil {
		var err error
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, req, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
		if err != nil {
			l.Error(err, "unable to get environment from cluster environment")
			return err
		}
		env = ce.Spec.EnvironmentName

	}

	for _, rs := range rsl.Items {
		// Check if the ServiceClassIdentity given in ServiceClaim is a subset of
		// ServiceClassIdentity given in the RegisteredService
		if checkSCISubset(sclaim.Spec.ServiceClassIdentity, rs.Spec.ServiceClassIdentity) &&
			(rs.Spec.Constraints == nil ||
				envtag.Match(env, rs.Spec.Constraints.Environments)) {
			registeredServiceFound = true
			registeredService = rs
			var err error
			count, err = r.extractServiceEndpointDefinition(ctx, req, rs, sclaim.Spec.ServiceEndpointDefinitionKeys, secret)
			if err != nil {
				l.Error(err, "unable to extract SED")
				return err
			}
			break
		}
	}

	if !registeredServiceFound {
		c := metav1.Condition{
			LastTransitionTime: metav1.NewTime(time.Now()),
			Type:               string(primazaiov1alpha1.ServiceClaimConditionReady),
			Status:             metav1.ConditionFalse,
			Reason:             constants.NoMatchingServiceFoundReason,
			Message:            "SCI is not matched",
		}
		meta.SetStatusCondition(&sclaim.Status.Conditions, c)

		sclaim.Status.State = "Pending"
		if err := r.Status().Update(ctx, &sclaim); err != nil {
			l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
			return err
		}

		return fmt.Errorf("SCI is not matched")
	}

	// if the number of SED keys is more than the number of secret data entries
	// that indicates one or more keys are missing
	if len(sclaim.Spec.ServiceEndpointDefinitionKeys) > count {
		c := metav1.Condition{
			LastTransitionTime: metav1.NewTime(time.Now()),
			Type:               string(primazaiov1alpha1.ServiceClaimConditionReady),
			Status:             metav1.ConditionFalse,
			Reason:             constants.NoMatchingServiceFoundReason,
			Message:            "key not available in the list of SEDs",
		}
		meta.SetStatusCondition(&sclaim.Status.Conditions, c)

		sclaim.Status.State = "Pending"
		if err := r.Status().Update(ctx, &sclaim); err != nil {
			l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
			return err
		}

		return fmt.Errorf("key not available in the list of SEDs")
	}

	// ServiceClassIdentity values are going to override
	// any values in the secret resource
	for _, sci := range sclaim.Spec.ServiceClassIdentity {
		secret.StringData[sci.Name] = sci.Value
	}

	// Update RegisteredService status to Claimed to avoid raise conditions
	if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateClaimed); err != nil {
		l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		return err
	}

	err := r.pushToClusterEnvironments(ctx, req, sclaim, secret)
	if err != nil {
		l.Error(err, "error pushing to cluster environments")
		// Update RegisteredService status back to Available
		if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateAvailable); err != nil {
			l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		}
		return client.IgnoreNotFound(err)
	}

	sclaim.Status.State = "Resolved"
	sclaim.Status.RegisteredService = registeredService.Name
	if err := r.Status().Update(ctx, &sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
		return err
	}

	return nil
}

func (r *ServiceClaimReconciler) pushToClusterEnvironments(
	ctx context.Context,
	req ctrl.Request,
	sclaim primazaiov1alpha1.ServiceClaim,
	secret *corev1.Secret,
) error {
	l := log.FromContext(ctx)
	errs := []error{}
	if sclaim.Spec.ApplicationClusterContext != nil {
		var err error
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, req, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
		if err != nil {
			l.Info("error getting ClusterEnvironment", "error", err)
			return err
		}
		cfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
		if err != nil {
			return err
		}
		if err = controlplane.PushServiceBinding(ctx, &sclaim, secret, r.Scheme, r.Client, &sclaim.Spec.ApplicationClusterContext.Namespace, ce.Spec.ApplicationNamespaces, cfg); err != nil {
			l.Error(err, "error pushing service binding", "serviceclaim", sclaim)
			errs = append(errs, err)
		}
	} else {
		var cel primazaiov1alpha1.ClusterEnvironmentList
		if err := r.List(ctx, &cel); err != nil {
			l.Info("error fetching ClusterEnvironmentList", "error", err)
			return client.IgnoreNotFound(err)
		}

		for _, ce := range cel.Items {
			cfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
			if err != nil {
				return err
			}
			// check if the ServiceClaim EnvironmentTag matches the EnvironmentName part of ClusterEnvironment
			if ce.Spec.EnvironmentName != sclaim.Spec.EnvironmentTag {
				l.Info("cluster environment is NOT matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
				continue
			}

			l.Info("cluster environment is matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
			if err = controlplane.PushServiceBinding(ctx, &sclaim, secret, r.Scheme, r.Client, nil, ce.Spec.ApplicationNamespaces, cfg); err != nil {
				errs = append(errs, err)
			}

		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (r *ServiceClaimReconciler) DeleteServiceBindingsAndSecret(
	ctx context.Context,
	req ctrl.Request,
	sclaim primazaiov1alpha1.ServiceClaim,
) error {
	l := log.FromContext(ctx)
	errs := []error{}

	if sclaim.Spec.ApplicationClusterContext != nil {
		var err error
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, req, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
		if err != nil {
			l.Info("error getting ClusterEnvironment", "error", err)
			return err
		}
		cli, err := clustercontext.CreateClient(ctx, r.Client, *ce, r.Scheme, r.Client.RESTMapper())
		if err != nil {
			return err
		}
		ns := []string{sclaim.Spec.ApplicationClusterContext.Namespace}
		if err = controlplane.DeleteServiceBindingAndSecretFromNamespaces(ctx, cli, sclaim, ns); err != nil {
			errs = append(errs, err)
		}
	} else {
		var cel primazaiov1alpha1.ClusterEnvironmentList
		if err := r.List(ctx, &cel); err != nil {
			l.Info("error fetching ClusterEnvironmentList", "error", err)
			return client.IgnoreNotFound(err)
		}

		for _, ce := range cel.Items {
			cli, err := clustercontext.CreateClient(ctx, r.Client, ce, r.Scheme, r.Client.RESTMapper())
			if err != nil {
				return err
			}
			// check if the ServiceClaim EnvironmentTag matches the EnvironmentName part of ClusterEnvironment
			if ce.Spec.EnvironmentName != sclaim.Spec.EnvironmentTag {
				l.Info("cluster environment is NOT matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
				continue
			}

			l.Info("cluster environment is matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
			if err = controlplane.DeleteServiceBindingAndSecretFromNamespaces(ctx, cli, sclaim, ce.Spec.ApplicationNamespaces); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceClaim{}).
		Complete(r)
}
