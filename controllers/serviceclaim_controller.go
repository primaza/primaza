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
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/uuid"
	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
	"github.com/primaza/primaza/pkg/primaza/constants"
	"github.com/primaza/primaza/pkg/primaza/controlplane"
)

// ClaimReconciler reconciles a Claim object
type ClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Mapper meta.RESTMapper
}

const ClaimFinalizer = "claims.primaza.io/finalizer"

func NewClaimReconciler(mgr ctrl.Manager) *ClaimReconciler {
	return &ClaimReconciler{
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
func (r *ClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("starting  service claim reconciliation")
	defer l.Info("reconciliation ended")

	var sclaim primazaiov1alpha1.Claim
	if err := r.Get(ctx, req.NamespacedName, &sclaim); err != nil {
		l.Info("unable to retrieve ServiceClaim", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Check if Service Claim is marked for deletion")
	if sclaim.HasDeletionTimestamp() {
		if controllerutil.ContainsFinalizer(&sclaim, ClaimFinalizer) {
			if err := r.processClaimMarkedForDeletion(ctx, req, sclaim); err != nil {
				return ctrl.Result{}, err
			}
			// Remove finalizer from service binding
			if finalizerBool := controllerutil.RemoveFinalizer(&sclaim, ClaimFinalizer); !finalizerBool {
				l.Error(errors.New("Finalizer not removed for service claim"), "Finalizer not removed for service claim")
				return ctrl.Result{}, errors.New("Finalizer not removed for service claim")
			}
			if err := r.Update(ctx, &sclaim); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// add finalizer if needed
	l.Info("Add Finalizer if needed")
	if controllerutil.AddFinalizer(&sclaim, ClaimFinalizer) {
		if err := r.Update(ctx, &sclaim); err != nil {
			return ctrl.Result{}, err
		}
	}

	l = l.WithValues("service-claim", sclaim.Name, "state", sclaim.Status.State)
	var err error
	switch sclaim.Status.State {
	case "":
		sclaim.Status.ClaimID = uuid.New().String()
		l.Info("reconciling new service claim")
		err = r.processClaim(ctx, req, sclaim)
	case primazaiov1alpha1.ClaimStateResolved:
		l.Info("reconciling Resolved service claim")
		err = r.processResolvedServiceClaim(ctx, sclaim)
	default:
		l.Info("reconciling Pending or marked for deletion service claim")
		err = r.processClaim(ctx, req, sclaim)
	}
	if err != nil {
		l.Error(err, "error processing ServiceClaim")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClaimReconciler) processClaim(ctx context.Context, req ctrl.Request, sclaim primazaiov1alpha1.Claim) error {
	l := log.FromContext(ctx)

	var rsl primazaiov1alpha1.RegisteredServiceList
	lo := client.ListOptions{Namespace: req.NamespacedName.Namespace}
	if err := r.List(ctx, &rsl, &lo); err != nil {
		l.Info("unable to retrieve RegisteredServiceList", "error", err)
		return client.IgnoreNotFound(err)
	}

	if err := r.processServiceClaim(ctx, rsl, sclaim); err != nil {
		l.Error(err, "error processing ServiceClaim")
		return err
	}
	return nil
}

func (r *ClaimReconciler) processClaimMarkedForDeletion(ctx context.Context, req ctrl.Request, sclaim primazaiov1alpha1.Claim) error {
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

func (r *ClaimReconciler) extractServiceEndpointDefinition(
	ctx context.Context,
	namespace string,
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
			if slices.Contains(sedKeys, sed.Name) {
				secret.StringData[sed.Name] = sed.Value
				count++
			}
		} else if k := sed.ValueFromSecret.Key; k != "" { // check value if the key is non-empty
			if slices.Contains(sedKeys, sed.Name) {
				sec := &corev1.Secret{}
				nn := types.NamespacedName{Namespace: namespace, Name: sed.ValueFromSecret.Name}
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

func (r *ClaimReconciler) changeServiceState(ctx context.Context, rs primazaiov1alpha1.RegisteredService, state primazaiov1alpha1.RegisteredServiceState) error {
	rs.Status.State = state
	if err := r.Status().Update(ctx, &rs); err != nil {
		return err
	}

	return nil
}

func (r *ClaimReconciler) getEnvironmentFromClusterEnvironment(
	ctx context.Context,
	namespace string,
	clusterEnvironmentName string) (*primazaiov1alpha1.ClusterEnvironment, error) {
	l := log.FromContext(ctx)
	ce := &primazaiov1alpha1.ClusterEnvironment{}
	objectKey := types.NamespacedName{Name: clusterEnvironmentName, Namespace: namespace}
	if err := r.Get(ctx, objectKey, ce); err != nil {
		l.Info("unable to retrieve ClusterEnvironment", "error", err)
		return nil, err
	}

	return ce, nil
}

func (r *ClaimReconciler) getServiceEndpointDefinition(
	ctx context.Context,
	sclaim primazaiov1alpha1.Claim,
	rs primazaiov1alpha1.RegisteredService,
) (*corev1.Secret, error) {
	l := log.FromContext(ctx)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sclaim.Name,
			Namespace: sclaim.Namespace,
		},
		StringData: map[string]string{},
	}

	// extract Service Endpoint Definition
	if _, err := r.extractServiceEndpointDefinition(
		ctx, sclaim.Namespace, rs, sclaim.Spec.ServiceEndpointDefinitionKeys, &secret); err != nil {
		l.Error(err, "error extracting ServiceEndpointDefinition",
			"registered-service", rs, "service-claim", sclaim)
		return nil, err
	}

	// ServiceClassIdentity values are going to override
	// any values in the secret resource
	for _, sci := range sclaim.Spec.ServiceClassIdentity {
		secret.StringData[sci.Name] = sci.Value
	}

	return &secret, nil
}

func (r *ClaimReconciler) processResolvedServiceClaim(
	ctx context.Context,
	sclaim primazaiov1alpha1.Claim) error {
	l := log.FromContext(ctx).WithValues("service-claim", sclaim)

	if sclaim.Status.RegisteredService == nil {
		err := fmt.Errorf("service claim %s's registered service name is not set", sclaim.Name)
		l.Error(err, "can not process resolved service claim")
		return err
	}

	// retrieve already bound RegisteredService
	var rs primazaiov1alpha1.RegisteredService
	k := types.NamespacedName{Name: sclaim.Status.RegisteredService.Name, Namespace: sclaim.Namespace}
	if err := r.Get(ctx, k, &rs, &client.GetOptions{}); err != nil {
		l.Info("error retrieving the RegisteredService", "error", err, "registered-service", k)
		return err
	}

	// bake the ServiceEndpointDefinition Secret
	secret, err := r.getServiceEndpointDefinition(ctx, sclaim, rs)
	if err != nil {
		l.Error(err, "error baking the ServiceEndpointDefinition", "registered-service", rs, "service-claim", sclaim)
		return err
	}

	sclaim.Status.State = primazaiov1alpha1.ClaimStatePending
	sclaim.Status.RegisteredService = &corev1.ObjectReference{
		Name: rs.Name,
		UID:  rs.UID,
	}
	if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
		l.Error(err, "error updating the ServiceClaim",
			"registered-service", rs, "service-claim", sclaim)
		return err
	}

	// Update RegisteredService status to Claimed to avoid raise conditions
	if err := r.changeServiceState(ctx, rs, primazaiov1alpha1.RegisteredServiceStateClaimed); err != nil {
		l.Error(err, "error updating the RegisteredService", "registered-service", rs, "service-claim", sclaim)
		return err
	}

	sclaim.Status.State = primazaiov1alpha1.ClaimStateResolved
	sclaim.Status.RegisteredService = &corev1.ObjectReference{
		Name: rs.Name,
		UID:  rs.UID,
	}
	if err := r.pushToClusterEnvironments(ctx, sclaim, secret); err != nil {
		l.Error(err,
			"error pushing the ServiceBinding and secret to the cluster environments",
			"registered-service", rs, "service-claim", sclaim)
		// Update RegisteredService status back to Available
		if err := r.changeServiceState(ctx, rs, primazaiov1alpha1.RegisteredServiceStateAvailable); err != nil {
			l.Error(err,
				"error updating the RegisteredService with details on failed push of Service Binding",
				"registered-service", rs, "service-claim", sclaim)
		}
		return err
	}

	if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
		l.Error(err, "error updating the ServiceClaim",
			"registered-service", rs, "service-claim", sclaim)
		return err
	}

	return nil
}

func (r *ClaimReconciler) updateServiceClaimStatus(ctx context.Context, sclaim *primazaiov1alpha1.Claim) error {
	l := log.FromContext(ctx).WithValues("service-claim", sclaim.Name, "status", sclaim.Status)

	l.Info("updating service-claim status in control plane")
	if err := r.Status().Update(ctx, sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim in Primaza's Control Plane")
		return err
	}

	l.Info("updating service-claim status in remote application namespace")
	if err := r.updateRemoteServiceClaimStatusIfNeeded(ctx, *sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim in Application Namespace")
		return err
	}

	return nil
}

func (r *ClaimReconciler) processServiceClaim(
	ctx context.Context,
	rsl primazaiov1alpha1.RegisteredServiceList,
	sclaim primazaiov1alpha1.Claim) error {
	l := log.FromContext(ctx)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sclaim.Name,
			Namespace: sclaim.Namespace,
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
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, sclaim.Namespace, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
		if err != nil {
			l.Error(err, "unable to get environment from cluster environment")
			return err
		}
		env = ce.Spec.EnvironmentName

	}

	for _, rs := range rsl.Items {
		// Check if the registered Service is Available
		if rs.Status.State != primazaiov1alpha1.RegisteredServiceStateAvailable {
			continue
		}

		// Check if the ServiceClassIdentity given in ServiceClaim is a subset of
		// ServiceClassIdentity given in the RegisteredService
		if checkSCISubset(sclaim.Spec.ServiceClassIdentity, rs.Spec.ServiceClassIdentity) &&
			(rs.Spec.Constraints == nil ||
				envtag.Match(env, rs.Spec.Constraints.Environments)) {
			registeredServiceFound = true
			registeredService = rs
			var err error
			count, err = r.extractServiceEndpointDefinition(
				ctx, sclaim.Namespace, rs, sclaim.Spec.ServiceEndpointDefinitionKeys, secret)
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

		sclaim.Status.State = primazaiov1alpha1.ClaimStatePending
		if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
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

		sclaim.Status.State = primazaiov1alpha1.ClaimStatePending
		if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
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

	sclaim.Status.State = primazaiov1alpha1.ClaimStatePending
	sclaim.Status.RegisteredService = &corev1.ObjectReference{
		Name: registeredService.Name,
		UID:  registeredService.UID,
	}
	if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
		return err
	}

	// Update RegisteredService status to Claimed to avoid raise conditions
	if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateClaimed); err != nil {
		l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		return err
	}

	sclaim.Status.State = primazaiov1alpha1.ClaimStateResolved
	sclaim.Status.RegisteredService = &corev1.ObjectReference{
		Name: registeredService.Name,
		UID:  registeredService.UID,
	}
	err := r.pushToClusterEnvironments(ctx, sclaim, secret)
	if err != nil {
		l.Error(err, "error pushing to cluster environments")
		// Update RegisteredService status back to Available
		if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateAvailable); err != nil {
			l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		}
		return client.IgnoreNotFound(err)
	}

	if err := r.updateServiceClaimStatus(ctx, &sclaim); err != nil {
		l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
		return err
	}

	return nil
}

func (r *ClaimReconciler) updateRemoteServiceClaimStatusIfNeeded(
	ctx context.Context,
	sclaim primazaiov1alpha1.Claim,
) error {
	l := log.FromContext(ctx).WithValues("service-claim", sclaim.Name)

	if sclaim.Spec.ApplicationClusterContext == nil {
		l.Info("Service claim is not Cluster-scoped, skipping status update")
		return nil
	}

	ce, err := r.getEnvironmentFromClusterEnvironment(ctx, sclaim.Namespace, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
	if err != nil {
		l.Info("error getting ClusterEnvironment", "error", err)
		return err
	}
	l.Info("retrieved ClusterEnvironment", "cluster-environment", ce.ObjectMeta)

	cfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
	if err != nil {
		return err
	}

	oc := client.Options{
		Scheme: r.Scheme,
		Mapper: r.Mapper,
	}
	cli, err := client.New(cfg, oc)
	if err != nil {
		return fmt.Errorf("error creating client for cluster environment %s: %w", ce.Name, err)
	}

	ans := sclaim.Spec.ApplicationClusterContext.Namespace
	otk := types.NamespacedName{Namespace: ans, Name: sclaim.Name}
	rsc := primazaiov1alpha1.ServiceClaim{}
	if err := cli.Get(ctx, otk, &rsc); err != nil {
		if sclaim.Status.State == primazaiov1alpha1.ClaimStatePending {
			l.Info("not updating service-claim status in remote application namespace as the claim is pending")
			return nil
		}
		return fmt.Errorf("error retrieving ServiceClaim from application namespace %s of cluster environment %s: %w", ans, ce.Name, err)
	}

	l = l.WithValues("remote-service-claim", rsc.Name, "application-namespace", rsc.Namespace)
	l.Info("retrieved remote service claim", "status", rsc.Status)

	l = l.WithValues("status", sclaim.Status)
	if rsc.Status.RegisteredService != sclaim.Status.RegisteredService ||
		string(rsc.Status.State) != string(sclaim.Status.State) {
		rsc.Status.RegisteredService = sclaim.Status.RegisteredService
		rsc.Status.State = v1alpha1.ServiceClaimState(sclaim.Status.State)
		if err := cli.Status().Update(ctx, &rsc); err != nil {
			l.Error(err, "error updating serviceclaim status")
			return fmt.Errorf("error updating ServiceClaim from application namespace %s of cluster environment %s: %w", ans, ce.Name, err)
		}
		l.Info("serviceclaim updated", "remote status", rsc.Status)
	} else {
		l.Info("no need to update serviceclaim status")
	}

	return nil
}

func (r *ClaimReconciler) pushToClusterEnvironments(
	ctx context.Context,
	sclaim primazaiov1alpha1.Claim,
	secret *corev1.Secret,
) error {
	l := log.FromContext(ctx)
	errs := []error{}
	if sclaim.Spec.ApplicationClusterContext != nil {
		var err error
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, sclaim.Namespace, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
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

func (r *ClaimReconciler) DeleteServiceBindingsAndSecret(
	ctx context.Context,
	req ctrl.Request,
	sclaim primazaiov1alpha1.Claim,
) error {
	l := log.FromContext(ctx)
	errs := []error{}

	if sclaim.Spec.ApplicationClusterContext != nil {
		var err error
		ce, err := r.getEnvironmentFromClusterEnvironment(ctx, sclaim.Namespace, sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName)
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
func (r *ClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	genPred := predicate.GenerationChangedPredicate{}
	reconcileOnRegisteredServiceUpdate := func(ctx context.Context, a client.Object) []reconcile.Request {
		l := log.FromContext(ctx)
		rs, ok := a.(*v1alpha1.RegisteredService)
		if !ok {
			l.Info("error parsing object to RegisteredService when mapping to ServiceClaim reconciliation trigger", "object", a)
			return []reconcile.Request{}
		}
		if rs.Status.State != `Claimed` {
			l.Info("Registered service is unclaimed, no service claim to reconcile", "registered-service", rs.Name)
			return []reconcile.Request{}
		}
		serviceclaims := &v1alpha1.ServiceClaimList{}
		opts := &client.ListOptions{}
		if err := r.List(ctx, serviceclaims, opts); err != nil {
			l.Error(err,
				"unable to list the ServiceClaims and reconcile for Registered Service Updates",
				"RegisteredService", rs.Name)
			return []reconcile.Request{}
		}
		for _, sc := range serviceclaims.Items {
			if sc.Status.State == primazaiov1alpha1.ServiceClaimStateResolved &&
				sc.Status.RegisteredService != nil && sc.Status.RegisteredService.UID == rs.UID {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{
					Namespace: sc.Namespace,
					Name:      sc.Name,
				}}}
			}
		}
		return []reconcile.Request{}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceClaim{}).
		Watches(&primazaiov1alpha1.RegisteredService{}, handler.EnqueueRequestsFromMapFunc(reconcileOnRegisteredServiceUpdate)).
		WithEventFilter(genPred).
		Complete(r)
}
