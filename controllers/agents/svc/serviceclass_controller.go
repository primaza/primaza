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

	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/jsonpath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
)

// This is the name of the secret that contains the information the service
// agents needs to write back registered services up to primaza.  It contains
// two keys: `kubeconfig`, a serialized kubeconfig for the upstream kubeconfig
// cluster, and `namespace`, the namespace to write registered services to
const PRIMAZA_CONTROLLER_REFERENCE string = "primaza-kubeconfig"

// ServiceClassReconciler reconciles a ServiceClass object
type ServiceClassReconciler struct {
	client.Client
	dynamic.Interface
	RemoteScheme *runtime.Scheme
	Mapper       meta.RESTMapper
}

const finalizer = "serviceclasses.primaza.io/finalizer"

func NewServiceClassReconciler(mgr ctrl.Manager) *ServiceClassReconciler {
	return &ServiceClassReconciler{
		Client:    mgr.GetClient(),
		Interface: dynamic.NewForConfigOrDie(mgr.GetConfig()),
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceClass object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ServiceClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcileLog := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	reconcileLog.Info("Reconciling service class")

	// first, get the service class
	serviceClass := v1alpha1.ServiceClass{}
	err := r.Get(ctx, req.NamespacedName, &serviceClass)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// service class was deleted on the control plane; we've got
			// nothing more to do here.
			return ctrl.Result{}, nil
		}
		// something went wrong, requeue
		reconcileLog.Error(err, "Failed to retrieve ServiceClass")
		return ctrl.Result{}, err
	}

	if err = r.setOwnerReference(ctx, &serviceClass, req.Namespace); err != nil {
		reconcileLog.Error(err, "Failed to set owner reference on ServiceClass", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	// next, get all the services that this service class controls
	services, err := r.GetResources(ctx, &serviceClass)
	if err != nil {
		reconcileLog.Error(err, "Failed to retrieve resources")
		return ctrl.Result{}, err
	}

	// then, write all the registered services up to the primaza cluster
	if serviceClass.DeletionTimestamp.IsZero() {
		handler := func(remote_client client.Client, rs v1alpha1.RegisteredService) error {
			op, err := controllerutil.CreateOrUpdate(ctx, remote_client, &rs, func() error { return nil })
			if err != nil {
				reconcileLog.Error(err, "Failed to create registered service",
					"service", rs.Name,
					"namespace", rs.Namespace)
			} else {
				reconcileLog.Info("Wrote registered service", "service", rs.Name, "namespace", rs.Namespace, "operation", op)
			}
			return err
		}

		// add a finalizer since we have deletion logic
		if controllerutil.AddFinalizer(&serviceClass, finalizer) {
			if err = r.Update(ctx, &serviceClass, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
		err = r.HandleRegisteredServices(ctx, &serviceClass, *services, handler)
		if err != nil {
			reconcileLog.Error(err, "Failed to write registered services")
			// fallthrough: we still want to write the service class status field
		}
	} else if controllerutil.ContainsFinalizer(&serviceClass, finalizer) {
		handler := func(remote_client client.Client, rs v1alpha1.RegisteredService) error {
			if err := remote_client.Delete(ctx, &rs); err != nil {
				if apierrors.IsNotFound(err) {
					// we tried to delete an object that doesn't exist, so
					return nil
				}
				reconcileLog.Error(err, "Failed to delete registered service", "namespace", rs.Namespace)
			}
			return err
		}

		err = r.HandleRegisteredServices(ctx, &serviceClass, *services, handler)
		if err != nil {
			reconcileLog.Error(err, "Failed to delete registered services")
			return ctrl.Result{}, err
		}

		// remove the finalizer so we don't requeue
		if controllerutil.RemoveFinalizer(&serviceClass, finalizer) {
			if err = r.Update(ctx, &serviceClass, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// nothing to do
		return ctrl.Result{}, nil
	}

	// finally, write the status of the service class
	err = r.Client.Status().Update(ctx, &serviceClass)
	if err != nil {
		reconcileLog.Error(err, "Failed to write service class status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ServiceClassReconciler) GetResources(ctx context.Context, serviceClass *v1alpha1.ServiceClass) (*unstructured.UnstructuredList, error) {
	typemeta := metav1.TypeMeta{
		Kind:       serviceClass.Spec.Resource.Kind,
		APIVersion: serviceClass.Spec.Resource.APIVersion,
	}
	gvk := typemeta.GroupVersionKind()
	mapping, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	services, err := r.Interface.Resource(mapping.Resource).
		Namespace(serviceClass.Namespace).
		List(ctx, metav1.ListOptions{})

	if err != nil || services == nil {
		return nil, err
	}

	return services, nil
}

type HandleFunc func(client.Client, v1alpha1.RegisteredService) error

func (r *ServiceClassReconciler) HandleRegisteredServices(ctx context.Context, serviceClass *v1alpha1.ServiceClass, services unstructured.UnstructuredList, handleFunc HandleFunc) error {
	l := log.FromContext(ctx)
	mappings := map[string]*jsonpath.JSONPath{}
	for _, mapping := range serviceClass.Spec.Resource.ServiceEndpointDefinitionMapping {
		path := jsonpath.New("")
		err := path.Parse(fmt.Sprintf("{%s}", mapping.JsonPath))
		if err != nil {
			return err
		}
		mappings[mapping.Name] = path
	}

	config, remote_namespace, err := r.getPrimazaKubeconfig(ctx, serviceClass.Namespace)
	if err != nil {
		return err
	}
	l.Info("remote cluster", "address", config.Host)

	// TODO(sadlerap): move TestConnection from `workercluster` into a more
	// general-purpose package
	status := workercluster.TestConnection(ctx, config)
	state := metav1.ConditionUnknown
	if status.State == v1alpha1.ClusterEnvironmentStateOnline {
		state = metav1.ConditionTrue
	} else if status.State == v1alpha1.ClusterEnvironmentStateOffline {
		state = metav1.ConditionFalse
	}
    meta.SetStatusCondition(&serviceClass.Status.Conditions, metav1.Condition{
		Type:               "Connection",
		Message:            status.Message,
		Reason:             string(status.Reason),
		Status:             state,
	})
	if status.State == v1alpha1.ClusterEnvironmentStateOffline {
		return fmt.Errorf("Failed to connect to cluster")
	}

	remote_client, err := client.New(config, client.Options{
		Scheme: r.Client.Scheme(),
		Mapper: r.Mapper,
	})
	if err != nil {
		return err
	}

	var errorList []error
	for _, data := range services.Items {
		sedMappings, err := LookupServiceEndpointDescriptor(mappings, data)
		if err != nil {
			l.Error(err, "Failed to lookup service endpoint descriptor values",
				"name", data.GetName(),
				"namespace", data.GetNamespace(),
				"gvk", data.GroupVersionKind())
		}

		rs := v1alpha1.RegisteredService{
			ObjectMeta: metav1.ObjectMeta{
				// FIXME(sadlerap): this could cause naming conflicts; we need
				// to take into account the type of resource somehow.
				Name:      data.GetName(),
				Namespace: remote_namespace,
			},
			Spec: v1alpha1.RegisteredServiceSpec{
				ServiceEndpointDefinition: sedMappings,
				ServiceClassIdentity:      serviceClass.Spec.ServiceClassIdentity,
				HealthCheck:               serviceClass.Spec.HealthCheck,
			},
		}

		if serviceClass.Spec.Constraints != nil {
			rs.Spec.Constraints = &v1alpha1.RegisteredServiceConstraints{
				Environments: serviceClass.Spec.Constraints.Environments,
			}
		}

		if err = handleFunc(remote_client, rs); err != nil {
			errorList = append(errorList, err)
		}
	}

	return errors.Join(errorList...)
}

func LookupServiceEndpointDescriptor(mappings map[string]*jsonpath.JSONPath, service unstructured.Unstructured) ([]v1alpha1.ServiceEndpointDefinitionItem, error) {
	var sedMappings []v1alpha1.ServiceEndpointDefinitionItem
	for key, jsonPath := range mappings {
		results, err := jsonPath.FindResults(service.Object)
		if err != nil {
			return nil, err
		}
		if len(results) == 1 && len(results[0]) == 1 {
			value := fmt.Sprintf("%v", results[0][0])
			sedMappings = append(sedMappings, v1alpha1.ServiceEndpointDefinitionItem{
				Name:  key,
				Value: value,
			})
		} else {
			return nil, fmt.Errorf("jsonPath lookup into resource returned multiple results: %v", results)
		}
	}

	return sedMappings, nil
}

func (r *ServiceClassReconciler) getPrimazaKubeconfig(ctx context.Context, namespace string) (*rest.Config, string, error) {
	// TODO(sadlerap): can we use the functionality in GetClusterRESTConfig
	// from pkg/primaza/clustercontext to do de-duplicate this?
	s := v1.Secret{}
	k := client.ObjectKey{Namespace: namespace, Name: PRIMAZA_CONTROLLER_REFERENCE}
	if err := r.Get(ctx, k, &s); err != nil {
		return nil, "", err
	}
	if _, found := s.Data["kubeconfig"]; !found {
		return nil, "", fmt.Errorf("Field \"kubeconfig\" field in secret %s:%s does not exist", s.Name, s.Namespace)
	}

	if _, found := s.Data["namespace"]; !found {
		return nil, "", fmt.Errorf("Field \"namespace\" field in secret %s:%s does not exist", s.Name, s.Namespace)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(s.Data["kubeconfig"])
	if err != nil {
		return nil, "", err
	}
	return restConfig, string(s.Data["namespace"]), nil
}

func (r *ServiceClassReconciler) setOwnerReference(ctx context.Context, scclass *v1alpha1.ServiceClass, namespace string) error {
	reconcileLog := log.FromContext(ctx)
	objKey := client.ObjectKey{
		Name:      "primaza-controller-agentsvc",
		Namespace: namespace,
	}
	var agentsvcdeployment appsv1.Deployment
	if err := r.Get(ctx, objKey, &agentsvcdeployment); err != nil {
		reconcileLog.Error(err, "unable to retrieve agent svc deployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests
		return client.IgnoreNotFound(err)
	}
	if err := ctrl.SetControllerReference(&agentsvcdeployment, scclass, r.Client.Scheme()); err != nil {
		return err
	}
	if err := r.Update(ctx, scclass); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceClass{}).
		Complete(r)
}
