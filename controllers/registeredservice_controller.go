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

	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
)

// RegisteredServiceReconciler reconciles a RegisteredService object
type RegisteredServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func ServiceInCatalog(sc primazaiov1alpha1.ServiceCatalog, serviceName string) int {
	for i, service := range sc.Spec.Services {
		if service.Name == serviceName {
			return i
		}
	}
	return -1
}

func (r *RegisteredServiceReconciler) getServiceCatalogs(ctx context.Context, namespace string) (primazaiov1alpha1.ServiceCatalogList, error) {
	log := log.FromContext(ctx)
	var cl primazaiov1alpha1.ServiceCatalogList
	lo := client.ListOptions{Namespace: namespace}
	if err := r.List(ctx, &cl, &lo); err != nil {
		log.Info("Unable to retrieve ServiceCatalogList", "error", err)
		return cl, err
	}

	return cl, nil
}

func (r *RegisteredServiceReconciler) removeServiceFromCatalogs(ctx context.Context, namespace string, serviceName string) error {
	log := log.FromContext(ctx)
	catalogs, err := r.getServiceCatalogs(ctx, namespace)
	if err != nil {
		log.Error(err, "Error found getting list of ServiceCatalog")
		return err
	}

	var errs []error
	for _, sc := range catalogs.Items {
		err = r.removeServiceFromCatalog(ctx, sc, namespace, serviceName)
		if err != nil {
			log.Error(err, "Error found removing RegisteredService to ServiceCatalog")
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (r *RegisteredServiceReconciler) removeServiceFromCatalog(ctx context.Context, sc primazaiov1alpha1.ServiceCatalog, namespace string, serviceName string) error {
	log := log.FromContext(ctx)

	si := ServiceInCatalog(sc, serviceName)

	if si == -1 {
		log.Info("No catalog entry found")
		return nil
	}

	sc.Spec.Services = append(sc.Spec.Services[:si], sc.Spec.Services[si+1:]...)
	log.Info("Updating Service Catalog")
	if err := r.Update(ctx, &sc); err != nil {
		// Service Catalog update failed
		log.Error(err, "Error found updating ServiceCatalog")
		return err
	}

	log.Info("Removed RegisteredService from ServiceCatalog", "RegisteredService", serviceName, "ServiceCatalog", sc.Name)
	return nil
}

func (r *RegisteredServiceReconciler) reconcileCatalogs(ctx context.Context, rs primazaiov1alpha1.RegisteredService) error {
	log := log.FromContext(ctx)
	catalogs, err := r.getServiceCatalogs(ctx, rs.Namespace)
	if err != nil {
		log.Error(err, "Error found getting list of ServiceCatalog")
		return err
	}

	var errs []error
	for _, sc := range catalogs.Items {
		if envtag.Match(sc.Name, rs.Spec.GetEnvironmentConstraints()) {
			log.Info("Constraint matched or no constraints, reconciling catalog")
			err = r.addServiceToCatalog(ctx, sc, rs)
			if err != nil {
				log.Error(err, "Error found adding RegisteredService to ServiceCatalog")
				errs = append(errs, err)
			}
			log.Info("Added RegisteredService to ServiceCatalog", "RegisteredService", rs.Name, "ServiceCatalog", sc.Name)
		} else {
			log.Info("Constraint mismatched, reconciling catalog")
			err = r.removeServiceFromCatalog(ctx, sc, rs.Namespace, rs.Name)
			if err != nil {
				log.Error(err, "Error found removing RegisteredService from ServiceCatalog")
				errs = append(errs, err)
			}

		}
	}

	return errors.Join(errs...)
}

func (r *RegisteredServiceReconciler) addServiceToCatalog(ctx context.Context, sc primazaiov1alpha1.ServiceCatalog, rs primazaiov1alpha1.RegisteredService) error {
	log := log.FromContext(ctx)

	// Extracting Keys of SED
	sedKeys := make([]string, 0, len(rs.Spec.ServiceEndpointDefinition))
	for i := 0; i < len(rs.Spec.ServiceEndpointDefinition); i++ {
		sedKeys = append(sedKeys, rs.Spec.ServiceEndpointDefinition[i].Name)
	}

	// Initializing Service Catalog Service
	scs := primazaiov1alpha1.ServiceCatalogService{
		Name:                          rs.Name,
		ServiceClassIdentity:          rs.Spec.ServiceClassIdentity,
		ServiceEndpointDefinitionKeys: sedKeys,
	}

	if ServiceInCatalog(sc, scs.Name) == -1 {
		log.Info("Updating Service Catalog")
		sc.Spec.Services = append(sc.Spec.Services, scs)
		err := r.Update(ctx, &sc)
		if err != nil {
			// Service Catalog update failed
			return err
		}

	}

	return nil
}

//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RegisteredService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *RegisteredServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var rs primazaiov1alpha1.RegisteredService
	err := r.Client.Get(ctx, req.NamespacedName, &rs)
	if err != nil && k8errors.IsNotFound(err) {
		log.Info("Registered Service not found, handling delete event")
		err = r.removeServiceFromCatalogs(ctx, req.NamespacedName.Namespace, req.Name)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error removing service from ServiceCatalog")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil

	} else if err != nil {
		log.Error(err, "Error fetching RegisteredService")
		return ctrl.Result{}, err
	}

	if rs.Status == (primazaiov1alpha1.RegisteredServiceStatus{}) {
		rs.Status = primazaiov1alpha1.RegisteredServiceStatus{
			State: primazaiov1alpha1.RegisteredServiceStateAvailable,
		}
		log.Info("Updating status of RegisteredService")
		err = r.Status().Update(ctx, &rs)
		if err != nil {
			log.Error(err, "RegisteredService Status Failed")
			return ctrl.Result{}, err
		}

		err = r.reconcileCatalogs(ctx, rs)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error adding service to ServiceCatalog")
			return ctrl.Result{}, err
		}

	} else if rs.Status.State == primazaiov1alpha1.RegisteredServiceStateAvailable {
		err = r.reconcileCatalogs(ctx, rs)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error adding service to ServiceCatalog")
			return ctrl.Result{}, err
		}
	} else if rs.Status.State == primazaiov1alpha1.RegisteredServiceStateClaimed ||
		rs.Status.State == primazaiov1alpha1.RegisteredServiceStateUnreachable {

		err = r.removeServiceFromCatalogs(ctx, req.NamespacedName.Namespace, req.Name)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error removing service from ServiceCatalog")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegisteredServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.RegisteredService{}).
		Complete(r)
}
