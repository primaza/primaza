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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
	"github.com/primaza/primaza/pkg/primaza/controlplane"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
	"github.com/primaza/primaza/pkg/slices"
)

type namespaceType string

func (t namespaceType) permissionRequiredReason() string {
	return fmt.Sprintf("%sNamespacePermissionsRequired", t)
}

const (
	clusterEnvironmentFinalizer = "clusterenvironment.primaza.io/finalizer"

	applicationNamespaceType namespaceType = "Application"
	serviceNamespaceType     namespaceType = "Service"

	PermissionsGrantedReason    = "PermissionsGranted"
	ClientCreationErrorReason   = "ClientCreationError"
	PermissionsNotGrantedReason = "PermissionsNotGranted"
)

// ClusterEnvironmentReconciler reconciles a ClusterEnvironment object
type ClusterEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,namespace=system,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=clusterenvironments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=clusterenvironments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=clusterenvironments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterEnvironment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ClusterEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// fetch the cluster environment
	ce := &primazaiov1alpha1.ClusterEnvironment{}
	if err := r.Client.Get(ctx, req.NamespacedName, ce); err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("error fetching ClusterEnvironment (deleted)", "error", err)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		l.Error(err, "Failed to get ClusterEnvironment")
		return ctrl.Result{}, err
	}

	// check if instance is marked to be deleted
	if ce.HasDeletionTimestamp() {
		if controllerutil.ContainsFinalizer(ce, clusterEnvironmentFinalizer) {
			// run finalizer
			if err := r.finalizeClusterEnvironment(ctx, ce); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer from cluster environment
			controllerutil.RemoveFinalizer(ce, clusterEnvironmentFinalizer)
			if err := r.Update(ctx, ce); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// add finalizer if needed
	if !controllerutil.ContainsFinalizer(ce, clusterEnvironmentFinalizer) {
		controllerutil.AddFinalizer(ce, clusterEnvironmentFinalizer)
		if err := r.Update(ctx, ce); err != nil {
			return ctrl.Result{}, err
		}
	}

	// get cluster config
	cfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
	if err != nil {
		if errors.Is(err, clustercontext.ErrSecretNotFound) {
			c := workercluster.ConnectionStatus{
				State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
				Reason:  ClientCreationErrorReason,
				Message: fmt.Sprintf("error creating the client: %s", err),
			}
			r.updateClusterEnvironmentStatus(ctx, ce, c)
			if err := r.Client.Status().Update(ctx, ce); err != nil {
				l.Error(err, "error updating cluster environment status", "status", ce.Status)
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, err
	}

	// test connection
	if err := r.testConnection(ctx, cfg, ce); err != nil {
		return ctrl.Result{}, err
	}

	// test permissions
	fann, fsnn, err := r.testNamespacesPermissions(ctx, cfg, ce)
	if err != nil {
		return ctrl.Result{}, err
	}

	// reconcile namespaces
	l.Info("reconciling namespaces",
		"application namespaces", ce.Spec.ApplicationNamespaces,
		"service namespaces", ce.Spec.ServiceNamespaces,
		"failed application namespaces (won't reconcile)", fann,
		"failed service namespaces (won't reconcile)", fsnn)
	errns := r.reconcileNamespaces(ctx, cfg, ce, fann, fsnn)
	errsns := r.reconcileServiceNamespaces(ctx, cfg, ce, fsnn)
	errans := r.reconcileApplicationNamespaces(ctx, cfg, ce, fann)
	if err := errors.Join(errns, errsns, errans); err != nil {
		l.Error(err, "error reconciling namespaces")
		return ctrl.Result{}, err
	}
	l.Info("namespaces reconciled")

	if err := r.Client.Status().Update(ctx, ce); err != nil {
		l.Error(err, "error updating cluster environment status", "status", ce.Status)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterEnvironmentReconciler) testConnection(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment) error {
	cr := workercluster.TestConnection(ctx, cfg)
	r.updateClusterEnvironmentStatus(ctx, ce, cr)

	if cr.Reason != workercluster.ConnectionSuccessful {
		return fmt.Errorf("can not connect to target cluster")
	}

	return nil
}

// TODO: eventually move this logic in `pkg/primaza/controlplane`
func (r *ClusterEnvironmentReconciler) reconcileServiceNamespaces(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment, failedServiceNamespaces []string) error {
	serviceclassesList := primazaiov1alpha1.ServiceClassList{}
	if err := r.List(ctx, &serviceclassesList, &client.ListOptions{Namespace: ce.Namespace}); err != nil {
		return client.IgnoreNotFound(err)
	}

	var serviceclassFilteredList []primazaiov1alpha1.ServiceClass
	for _, serviceclass := range serviceclassesList.Items {
		if envtag.Match(ce.Spec.EnvironmentName, serviceclass.Spec.Constraints.Environments) {
			serviceclassFilteredList = append(serviceclassFilteredList, serviceclass)
		}
	}
	serviceNamespaces := slices.SubtractStr(ce.Spec.ServiceNamespaces, failedServiceNamespaces)

	errs := []error{}
	for _, serviceclass := range serviceclassFilteredList {
		cli, err := clustercontext.CreateClient(ctx, r.Client, *ce, r.Scheme, r.Client.RESTMapper())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := controlplane.PushServiceClassToNamespaces(ctx, cli, serviceclass, serviceNamespaces); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				errs = append(errs,
					fmt.Errorf("error pushing service class '%s' to cluster environment '%s': %w", serviceclass.Name, ce.Name, err))
			}
		}
	}

	return errors.Join(errs...)
}

// TODO: eventually move this logic in `pkg/primaza/controlplane`
func (r *ClusterEnvironmentReconciler) reconcileServiceBindingApplicationNamespaces(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment, applicationNamespaces []string) error {
	errs := []error{}
	l := log.FromContext(ctx)
	serviceclaimsList := primazaiov1alpha1.ServiceClaimList{}
	if err := r.List(ctx, &serviceclaimsList, &client.ListOptions{Namespace: ce.Namespace}); err != nil {
		return client.IgnoreNotFound(err)
	}
	var serviceclaimFilteredList []primazaiov1alpha1.ServiceClaim
	for _, serviceclaim := range serviceclaimsList.Items {
		if ce.Spec.EnvironmentName == serviceclaim.Spec.EnvironmentTag {
			serviceclaimFilteredList = append(serviceclaimFilteredList, serviceclaim)
		}
	}
	for index := range serviceclaimFilteredList {
		sclaim := serviceclaimFilteredList[index]
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sclaim.Name,
				Namespace: sclaim.Namespace,
			},
			StringData: map[string]string{},
		}
		// ServiceClassIdentity values are going to override
		// any values in the secret resource
		for _, sci := range sclaim.Spec.ServiceClassIdentity {
			secret.StringData[sci.Name] = sci.Value
		}
		if sclaim.Spec.EnvironmentTag == "" {
			if sclaim.Spec.ApplicationClusterContext != nil && ce.Name == sclaim.Spec.ApplicationClusterContext.ClusterEnvironmentName {
				if err := controlplane.PushServiceBinding(ctx, &sclaim, secret, r.Scheme, r.Client, &sclaim.Spec.ApplicationClusterContext.Namespace, applicationNamespaces, cfg); err != nil {
					errs = append(errs, err)
				}
			}
		} else {
			// check if the ServiceClaim EnvironmentTag matches the EnvironmentName part of ClusterEnvironment
			if ce.Spec.EnvironmentName != sclaim.Spec.EnvironmentTag {
				l.Info("cluster environment is NOT matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
				continue
			}

			l.Info("cluster environment is matching environment", "cluster environment", ce, "environment tag", sclaim.Spec.EnvironmentTag)
			if err := controlplane.PushServiceBinding(ctx, &sclaim, secret, r.Scheme, r.Client, nil, applicationNamespaces, cfg); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (r *ClusterEnvironmentReconciler) reconcileServiceCatalogApplicationNamespaces(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment, applicationNamespaces []string) error {
	servicecatalog := primazaiov1alpha1.ServiceCatalog{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ce.Namespace, Name: ce.Spec.EnvironmentName}, &servicecatalog); apierrors.IsNotFound(err) {
		if err := r.CreateServiceCatalog(ctx, ce); err != nil {
			return err
		}
	}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ce.Namespace, Name: ce.Spec.EnvironmentName}, &servicecatalog); err != nil {
		return err
	}
	if err := controlplane.PushServiceCatalogToApplicationNamespaces(ctx, servicecatalog, r.Scheme, r.Client, applicationNamespaces, cfg); err != nil {
		return err
	}
	return nil
}

func (r *ClusterEnvironmentReconciler) reconcileApplicationNamespaces(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment, failedApplicationNamespaces []string) error {

	nns := slices.SubtractStr(ce.Spec.ApplicationNamespaces, failedApplicationNamespaces)
	errcm := r.reconcileServiceBindingApplicationNamespaces(ctx, cfg, ce, nns)
	errct := r.reconcileServiceCatalogApplicationNamespaces(ctx, cfg, ce, nns)
	return errors.Join(errcm, errct)
}

func (r *ClusterEnvironmentReconciler) testNamespacesPermissions(ctx context.Context, cfg *rest.Config, ce *primazaiov1alpha1.ClusterEnvironment) ([]string, []string, error) {
	// check application namespaces permissions
	apc := controlplane.NewAgentAppPermissionsChecker(cfg)
	ansp, err := r.testTypedNamespacesPermissions(ctx, ce, applicationNamespaceType, apc, ce.Spec.ApplicationNamespaces)
	if err != nil {
		return nil, nil, err
	}

	// check service namespaces permissions
	spc := controlplane.NewAgentSvcPermissionsChecker(cfg)
	snsp, err := r.testTypedNamespacesPermissions(ctx, ce, serviceNamespaceType, spc, ce.Spec.ServiceNamespaces)
	if err != nil {
		return nil, nil, err
	}

	// set status to Partial if at least one namespace is not configured correctly
	if len(ansp) > 0 || len(snsp) > 0 {
		ce.Status.State = primazaiov1alpha1.ClusterEnvironmentStatePartial
	}

	return ansp, snsp, nil
}

func (r *ClusterEnvironmentReconciler) testTypedNamespacesPermissions(
	ctx context.Context,
	ce *primazaiov1alpha1.ClusterEnvironment,
	nsType namespaceType,
	pc controlplane.AgentPermissionsChecker,
	namespaces []string) ([]string, error) {
	l := log.FromContext(ctx)

	pr, err := pc.TestPermissions(ctx, namespaces)
	if err != nil {
		return nil, err
	}

	failed := []string{}
	for ns, rp := range pr {
		if !rp.AllSatisfied() {
			failed = append(failed, ns)
			l.Info("namespace permission test failed", "namespace type", nsType, "namespace", ns, "report", rp)
		}
	}

	co := r.buildPermissionCondition(ctx, nsType, failed)
	meta.SetStatusCondition(&ce.Status.Conditions, co)

	return failed, nil
}

func (r *ClusterEnvironmentReconciler) buildPermissionCondition(ctx context.Context, nsType namespaceType, failedNamespaces []string) metav1.Condition {
	if len(failedNamespaces) > 0 {
		msg := fmt.Sprintf("namespaces missing required permissions: %v", failedNamespaces)

		return metav1.Condition{
			Type:    nsType.permissionRequiredReason(),
			Status:  metav1.ConditionTrue,
			Reason:  PermissionsNotGrantedReason,
			Message: msg,
		}
	}

	return metav1.Condition{
		Type:    nsType.permissionRequiredReason(),
		Status:  metav1.ConditionFalse,
		Reason:  PermissionsGrantedReason,
		Message: "all required permissions are granted",
	}
}

func (r *ClusterEnvironmentReconciler) reconcileNamespaces(
	ctx context.Context,
	cfg *rest.Config,
	ce *primazaiov1alpha1.ClusterEnvironment,
	failedApplicationNamespaces, failedServiceNamespaces []string) error {
	ans := slices.SubtractStr(ce.Spec.ApplicationNamespaces, failedApplicationNamespaces)
	sns := slices.SubtractStr(ce.Spec.ServiceNamespaces, failedServiceNamespaces)

	s := controlplane.ClusterEnvironmentState{
		Name:                  ce.Name,
		Namespace:             ce.Namespace,
		ClusterConfig:         cfg,
		ApplicationNamespaces: ans,
		ServiceNamespaces:     sns,
	}

	nr, err := controlplane.NewNamespaceReconciler(s)
	if err != nil {
		return err
	}

	if err := nr.ReconcileNamespaces(ctx); err != nil {
		return err
	}
	return nil
}

func (r *ClusterEnvironmentReconciler) updateClusterEnvironmentStatus(ctx context.Context, ce *primazaiov1alpha1.ClusterEnvironment, cs workercluster.ConnectionStatus) {
	l := log.FromContext(ctx)

	l.Info("updating cluster environment status", "clusterenvironment", ce.GetName(), "connection status", cs)
	ce.Status.State = cs.State
	meta.SetStatusCondition(&ce.Status.Conditions, cs.Condition())
}

func (r *ClusterEnvironmentReconciler) getRelatedClusterEnvironments(ctx context.Context, namespace string, envname string) ([]primazaiov1alpha1.ClusterEnvironment, error) {
	cee := primazaiov1alpha1.ClusterEnvironmentList{}
	if err := r.List(ctx, &cee, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	ff := r.filterClusterEnvironments(envname, cee.Items)
	return ff, nil
}

func (r *ClusterEnvironmentReconciler) filterClusterEnvironments(
	environmentName string,
	clusterEnvironments []primazaiov1alpha1.ClusterEnvironment) []primazaiov1alpha1.ClusterEnvironment {

	cee := []primazaiov1alpha1.ClusterEnvironment{}
	for _, ce := range clusterEnvironments {
		if ce.Spec.EnvironmentName == environmentName && !ce.HasDeletionTimestamp() {
			cee = append(cee, ce)
		}
	}

	return cee
}

func (r *ClusterEnvironmentReconciler) finalizeClusterEnvironment(ctx context.Context, ce *primazaiov1alpha1.ClusterEnvironment) error {
	var err []error
	errnamespace := r.finalizeClusterEnvironmentInNamespaces(ctx, ce)
	errcatalog := r.removeServiceCatalogOnDeletedClusterEnvironment(ctx, ce)
	err = append(err, errnamespace, errcatalog)
	return errors.Join(err...)
}

func (r *ClusterEnvironmentReconciler) removeServiceCatalogOnDeletedClusterEnvironment(ctx context.Context, ce *primazaiov1alpha1.ClusterEnvironment) error {

	ff, err := r.getRelatedClusterEnvironments(ctx, ce.Namespace, ce.Spec.EnvironmentName)
	if err != nil {
		return err
	}
	if len(ff) == 0 {
		servicecatalog := &primazaiov1alpha1.ServiceCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ce.Spec.EnvironmentName,
				Namespace: ce.Namespace,
			},
		}

		if err := r.Delete(ctx, servicecatalog, &client.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}

	return nil
}

func (r *ClusterEnvironmentReconciler) CreateServiceCatalog(ctx context.Context, ce *primazaiov1alpha1.ClusterEnvironment) error {
	l := log.FromContext(ctx)
	l.Info("Service catalog not found in cluster environment")

	var rsl primazaiov1alpha1.RegisteredServiceList
	lo := client.ListOptions{Namespace: ce.Namespace}
	if err := r.Client.List(ctx, &rsl, &lo); err != nil {
		return err
	}
	var scs []primazaiov1alpha1.ServiceCatalogService
	for _, rs := range rsl.Items {
		if rs.Status.State == primazaiov1alpha1.RegisteredServiceStateAvailable &&
			(rs.Spec.Constraints == nil || envtag.Match(ce.Spec.EnvironmentName, rs.Spec.Constraints.Environments)) {
			// Extracting Keys of SED
			sedKeys := make([]string, 0, len(rs.Spec.ServiceEndpointDefinition))
			for i := 0; i < len(rs.Spec.ServiceEndpointDefinition); i++ {
				sedKeys = append(sedKeys, rs.Spec.ServiceEndpointDefinition[i].Name)
			}
			// Initializing Service Catalog Service
			serviceCatalogSvc := primazaiov1alpha1.ServiceCatalogService{
				Name:                          rs.Name,
				ServiceClassIdentity:          rs.Spec.ServiceClassIdentity,
				ServiceEndpointDefinitionKeys: sedKeys,
			}
			scs = append(scs, serviceCatalogSvc)
		}
	}
	serviceCatalog := primazaiov1alpha1.ServiceCatalog{
		ObjectMeta: v1.ObjectMeta{
			Name:      ce.Spec.EnvironmentName,
			Namespace: ce.Namespace,
		},
		Spec: primazaiov1alpha1.ServiceCatalogSpec{
			Services: scs,
		},
	}

	l.Info(ce.Spec.EnvironmentName)
	if err := r.Create(ctx, &serviceCatalog); !apierrors.IsAlreadyExists(err) {
		l.Error(err, "Failed to create service catalog")
		return err
	}
	return nil
}

func (r *ClusterEnvironmentReconciler) finalizeClusterEnvironmentInNamespaces(ctx context.Context, ce *primazaiov1alpha1.ClusterEnvironment) error {
	kcfg, err := clustercontext.GetClusterRESTConfig(ctx, r.Client, ce.Namespace, ce.Spec.ClusterContextSecret)
	if err != nil {
		return err
	}

	s := controlplane.ClusterEnvironmentState{
		Name:                  ce.Name,
		Namespace:             ce.Namespace,
		ClusterConfig:         kcfg,
		ApplicationNamespaces: []string{},
		ServiceNamespaces:     []string{},
	}

	nr, err := controlplane.NewNamespaceReconciler(s)
	if err != nil {
		return err
	}

	return nr.ReconcileNamespaces(ctx)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ClusterEnvironment{}).
		Complete(r)
}
