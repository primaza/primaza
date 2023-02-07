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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
)

// RegisteredServiceReconciler reconciles a RegisteredService object
type RegisteredServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=primaza.io,resources=registeredservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,resources=registeredservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,resources=registeredservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=primaza.io,resources=servicecatalogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,resources=servicecatalogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,resources=servicecatalogs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RegisteredService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *RegisteredServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var rs primazaiov1alpha1.RegisteredService
	if err := r.Client.Get(ctx, req.NamespacedName, &rs); err != nil {
		log.Error(err, "Error fetching RegiteredService")
		return ctrl.Result{}, err
	}

	// Add logic to test connectivity here with HealthCheck Image and Cmd
	//can_connect = r.checkConnection(ctx, rs)
	can_connect := true //Hardcoding for now. This will change based on Healthcheck logic

	log.Info("Modifying status of RegisteredService")

	var status primazaiov1alpha1.RegisteredServiceStatus
	if can_connect {
		status = primazaiov1alpha1.RegisteredServiceStatus{
			State: primazaiov1alpha1.RegisteredServiceStateAvailable,
		}

	} else {
		status = primazaiov1alpha1.RegisteredServiceStatus{
			State: primazaiov1alpha1.RegisteredServiceStateUnreachable,
		}
	}

	rs.Status = status

	log.Info("Updating status of RegisteredService")
	err := r.Status().Update(context.Background(), &rs)
	if err != nil {
		log.Error(err, "RegisteredService Status Failed")
		return ctrl.Result{}, err
	}

	if status.State == primazaiov1alpha1.RegisteredServiceStateAvailable {
		var sc primazaiov1alpha1.ServiceCatalog
		err = r.Get(ctx, types.NamespacedName{
			Name:      "primaza-service-catalog",
			Namespace: rs.Namespace,
		}, &sc)

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

		if err != nil && errors.IsNotFound(err) {
			log.Info("Service Catalog not found, creating new")
			// Initializing the Service Catalog
			serviceCatalog := primazaiov1alpha1.ServiceCatalog{
				TypeMeta: v1.TypeMeta{
					Kind:       "ServiceCatalog",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      "primaza-service-catalog",
					Namespace: rs.Namespace,
				},
				Spec: primazaiov1alpha1.ServiceCatalogSpec{
					Services: []primazaiov1alpha1.ServiceCatalogService{scs},
				},
			}

			// Create the Service Catalog
			err = r.Create(ctx, &serviceCatalog)
			if err != nil {
				// Service Catalog creation failed
				return ctrl.Result{}, err
			}

		} else if err != nil {
			// Error that isn't due to the ServiceCatalog not found
			return ctrl.Result{}, err
		} else {
			log.Info("Updating Service Catalog")
			sc.Spec.Services = append(sc.Spec.Services, scs)
			err = r.Update(ctx, &sc)
			if err != nil {
				// Service Catalog update failed
				return ctrl.Result{}, err
			}

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
