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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/uuid"
	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	"github.com/primaza/primaza/pkg/slices"
)

// ServiceClaimReconciler reconciles a ServiceClaim object
type ServiceClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=primaza.io,resources=serviceclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,resources=serviceclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,resources=serviceclaims/finalizers,verbs=update

// TODO: Remove this later once Primaza ServiceBinding is implemented.
//+kubebuilder:rbac:groups=servicebinding.io,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;create;update;patch;delete

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

	l.Info("starting reconciliation")

	var sclaim primazaiov1alpha1.ServiceClaim

	if err := r.Get(ctx, req.NamespacedName, &sclaim); err != nil {
		l.Info("unable to retrieve ServiceClaim", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var rsl primazaiov1alpha1.RegisteredServiceList
	lo := client.ListOptions{Namespace: req.NamespacedName.Namespace}
	if err := r.List(ctx, &rsl, &lo); err != nil {
		l.Info("unable to retrieve RegisteredServiceList", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err := r.processServiceClaim(ctx, req, rsl, sclaim)
	if err != nil {
		l.Error(err, "unable to process ServiceClaim")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
	SEDKeys []string,
	secret *corev1.Secret) (int, error) {
	l := log.FromContext(ctx)
	count := 0
	// loop over the ServiceEndpointDefinition array part of RegisteredService
	for _, sed := range rs.Spec.ServiceEndpointDefinition {
		// check if the value is non-empty
		if sed.Value != "" {
			// check if the ServiceEndpointDefinitionKeys part of ServiceClaim has the current
			// SED name in the RegisteredService
			if slices.ItemContains(SEDKeys, sed.Name) {
				secret.StringData[sed.Name] = sed.Value
				count = count + 1
			}
		} else if sed.ValueFromSecret.Key != "" { // check value if the key is non-empty
			if slices.ItemContains(SEDKeys, sed.Name) {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      sed.ValueFromSecret.Name,
						Namespace: req.NamespacedName.Namespace,
					},
				}
				if err := r.Get(ctx, req.NamespacedName, sec); err != nil {
					l.Info("unable to retrieve Secret", "error", err)
					return 0, client.IgnoreNotFound(err)
				}
				secret.StringData[sed.Name] = sec.StringData[sed.ValueFromSecret.Key]
				count = count + 1
			}
		}
	}
	return count, nil
}


func (r *ServiceClaimReconciler) changeServiceState( ctx context.Context, rs primazaiov1alpha1.RegisteredService, state string ) error {
	rs.Status.State = state
	if err := r.Status().Update(ctx, &rs); err != nil {
		return err
	}

	return nil
}


func (r *ServiceClaimReconciler) processServiceClaim(
	ctx context.Context,
	req ctrl.Request,
	rsl primazaiov1alpha1.RegisteredServiceList,
	sclaim primazaiov1alpha1.ServiceClaim, 
) error {
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
	for _, rs := range rsl.Items {
		// Check if the ServiceClassIdentity given in ServiceClaim is a subset of
		// ServiceClassIdentity given in the RegisteredService
		if checkSCISubset(sclaim.Spec.ServiceClassIdentity, rs.Spec.ServiceClassIdentity) &&
		    envtag.Match(sclaim.Spec.EnvironmentTag, rs.Spec.Constraints.Environments) {
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
		sclaim.Status.State = "Pending"
		if err := r.Status().Update(ctx, &sclaim); err != nil {
			l.Error(err, "unable to update the ServiceClaim", "ServiceClaim", sclaim)
			return err
		}

		return fmt.Errorf("Key not available in the list of SEDs")
	}

	// ServiceClassIdentity values are going to override
	// any values in the secret resource
	for _, sci := range sclaim.Spec.ServiceClassIdentity {
		secret.StringData[sci.Name] = sci.Value
	}


	var cel primazaiov1alpha1.ClusterEnvironmentList
	if err := r.List(ctx, &cel); err != nil {
		l.Info("error fetching ClusterEnvironmentList", "error", err)
		return client.IgnoreNotFound(err)
	}

	// Update RegisteredService status to Claimed to avoid raise conditions
	if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateClaimed); err != nil {
		l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		return err
	}

	err := r.pushToClusterEnvironments(ctx, req, cel, sclaim, secret)
	if err != nil {
		l.Error(err, "error pushing to cluster environments")
		// Update RegisteredService status back to Available
		if err := r.changeServiceState(ctx, registeredService, primazaiov1alpha1.RegisteredServiceStateAvailable); err != nil {
			l.Error(err, "unable to update the RegisteredService", "RegisteredService", registeredService)
		}
		return client.IgnoreNotFound(err)
	}

	sclaim.Status.State = "Resolved"
	sclaim.Status.ClaimID = uuid.New().String()
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
	cel primazaiov1alpha1.ClusterEnvironmentList,
	sclaim primazaiov1alpha1.ServiceClaim,
	secret *corev1.Secret,
) error {
	// loop over all ClusterEnvironment
	for _, ce := range cel.Items {
		// check if the ServiceClaim EnvironmentTag matches the EnvironmentName part of ClusterEnvironment
		if sclaim.Spec.EnvironmentTag != ce.Spec.EnvironmentName {
			continue
		}
		sn := ce.Spec.ClusterContextSecret
		k := client.ObjectKey{Namespace: ce.Namespace, Name: sn}
		var s corev1.Secret
		if err := r.Get(ctx, k, &s); err != nil {
			return err
		}

		kubeconfig := s.Data["kubeconfig"]
		cg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		if err != nil {
			return err
		}

		cs, err := kubernetes.NewForConfig(cg)
		if err != nil {
			return err
		}

		_, err = cs.CoreV1().Secrets(ce.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		dynamicClient, err := dynamic.NewForConfig(cg)
		if err != nil {
			return err
		}

		sb := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       "ServiceBinding",
				"apiVersion": "servicebinding.io/v1beta1",
				"metadata": map[string]interface{}{
					"name":      req.NamespacedName.Name,
					"namespace": req.NamespacedName.Namespace,
				},
				"spec": map[string]interface{}{
					"service": map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Secret",
						"name":       req.NamespacedName.Name,
					},
					"workload": map[string]interface{}{
						"apiVersion": sclaim.Spec.Application.APIVersion,
						"kind":       sclaim.Spec.Application.Kind,
						"selector": map[string]interface{}{
							"matchLabels": sclaim.Spec.Application.Selector.MatchLabels,
						},
					},
				},
			}}
		gvr := schema.GroupVersionResource{
			Group:    "servicebinding.io",
			Version:  "v1beta1",
			Resource: "servicebindings",
		}
		_, err = dynamicClient.Resource(gvr).Namespace(req.NamespacedName.Namespace).Create(ctx, sb, metav1.CreateOptions{})
		if err != nil {
			return err
		}

	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceClaim{}).
		Complete(r)
}
