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

	"go.uber.org/atomic"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/constants"
	"github.com/primaza/primaza/pkg/primaza/sed"
	"github.com/primaza/primaza/pkg/primaza/workercluster"
)

const finalizer = "serviceclasses.primaza.io/finalizer"

// ServiceClassReconciler reconciles a ServiceClass object
type ServiceClassReconciler struct {
	client.Client
	dynamic.Interface
	informers map[string]informer
}

type informer struct {
	informer   cache.SharedIndexInformer
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (i *informer) run() {
	i.informer.Run(i.ctx.Done())
}

func NewServiceClassReconciler(mgr ctrl.Manager) *ServiceClassReconciler {
	return &ServiceClassReconciler{
		Client:    mgr.GetClient(),
		Interface: dynamic.NewForConfigOrDie(mgr.GetConfig()),
		informers: make(map[string]informer, 0),
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

	// get the controller's deployment
	controller := appsv1.Deployment{}
	controllerRef := types.NamespacedName{Namespace: serviceClass.Namespace, Name: constants.ServiceAgentDeploymentName}
	if err = r.Get(ctx, controllerRef, &controller); err != nil {
		reconcileLog.Error(err, "Failed to retrieve controller reference")
		if apierrors.IsNotFound(err) {
			// FIXME(sadlerap): the deployment's been deleted, and the pod
			// we're running in is likely going to be deleted soon as well.  Do
			// we have a cleaner way of triggering our own shutdown?
			reconcileLog.Error(err, "can not find agent's deployment", "agent", constants.ServiceAgentDeploymentName)
			os.Exit(1)
		}
		return ctrl.Result{}, err
	}

	if err = r.setOwnerReference(ctx, &serviceClass, &controller); err != nil {
		reconcileLog.Error(err, "Failed to set owner reference on ServiceClass", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	if err = r.SetWatchersForResources(ctx, serviceClass); err != nil {
		reconcileLog.Error(err, "Failed to set watchers on ServiceClass resources ", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	// next, get all the services that this service class controls
	services, err := r.GetResources(ctx, &serviceClass)
	if err != nil {
		reconcileLog.Error(err, "Failed to retrieve resources")
		return ctrl.Result{}, err
	}
	// then, write all the registered services up to the primaza cluster
	errs := []error{}
	if serviceClass.DeletionTimestamp.IsZero() && controller.DeletionTimestamp.IsZero() {
		// add a finalizer since we have deletion logic
		if controllerutil.AddFinalizer(&serviceClass, finalizer) {
			if err = r.Update(ctx, &serviceClass, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, err
			}
		}

		err = r.HandleRegisteredServices(ctx, &serviceClass, *services, updateRegisteredService)
		if err != nil {
			reconcileLog.Error(err, "Failed to write registered services")
			// fallthrough: we still want to write the service class status field
			errs = append(errs, err)
		}
	} else if controllerutil.ContainsFinalizer(&serviceClass, finalizer) {
		// need to stop the informers if the service class is deleted
		if i, ok := r.informers[serviceClass.Name]; ok {
			i.cancelFunc()
			delete(r.informers, serviceClass.Name)
		}

		// act on the registered service
		err = r.HandleRegisteredServices(ctx, &serviceClass, *services, deleteRegisteredService)
		if err != nil {
			reconcileLog.Error(err, "Failed to delete registered services")
			errs = append(errs, err)
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
		errs = append(errs, err)
	}
	return ctrl.Result{}, errors.Join(errs...)
}

func updateRegisteredService(ctx context.Context, remote_client client.Client, rs v1alpha1.RegisteredService, secret *v1.Secret) []error {
	spec := rs.Spec
	reconcileLog := log.FromContext(ctx).WithValues("namespace", rs.Namespace, "name", rs.Name)
	op, err := controllerutil.CreateOrUpdate(ctx, remote_client, &rs, func() error {
		rs.Spec = spec
		return nil
	})
	if err != nil {
		reconcileLog.Error(err, "Failed to create registered service", "service", rs.Name, "namespace", rs.Namespace)
	} else {
		reconcileLog.Info("Wrote registered service", "service", rs.Name, "namespace", rs.Namespace, "operation", op)
	}
	errs := []error{err}
	if secret != nil {
		data := secret.StringData
		_, err := controllerutil.CreateOrUpdate(ctx, remote_client, secret, func() error {
			secret.StringData = data
			return controllerutil.SetOwnerReference(&rs, secret, remote_client.Scheme())
		})
		errs = append(errs, err)
	}
	return errs
}

func deleteRegisteredService(ctx context.Context, remote_client client.Client, rs v1alpha1.RegisteredService, secret *v1.Secret) []error {
	reconcileLog := log.FromContext(ctx).WithValues("namespace", rs.Namespace, "name", rs.Name)
	if err := remote_client.Delete(ctx, &rs); err != nil {
		if apierrors.IsNotFound(err) {
			// we tried to delete an object that doesn't exist, so
			return nil
		}
		reconcileLog.Error(err, "Failed to delete registered service", "namespace", rs.Namespace)
		return []error{err}
	}

	// we don't need to delete the secret, since the secret had the registered
	// service set as an owner
	return nil
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

type HandleFunc func(context.Context, client.Client, v1alpha1.RegisteredService, *v1.Secret) []error

func (r *ServiceClassReconciler) HandleRegisteredServices(ctx context.Context, serviceClass *v1alpha1.ServiceClass, services unstructured.UnstructuredList, handleFunc HandleFunc) error {
	l := log.FromContext(ctx)
	var err error

	config, remote_namespace, err := workercluster.GetPrimazaKubeconfig(ctx)
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
		Type:    "Connection",
		Message: status.Message,
		Reason:  string(status.Reason),
		Status:  state,
	})
	if status.State == v1alpha1.ClusterEnvironmentStateOffline {
		return fmt.Errorf("Failed to connect to cluster")
	}

	remote_client, err := client.New(config, client.Options{
		Scheme: r.Client.Scheme(),
		Mapper: r.Client.RESTMapper(),
	})
	if err != nil {
		return err
	}

	var errorList []error
	for _, data := range services.Items {
		var mappings []sed.SEDMapping
		if mappings, err = ServiceEndpointDefinitionMapping(r.Client, data, *serviceClass); err != nil {
			return err
		}

		var rs v1alpha1.RegisteredService
		var err error
		var secret *v1.Secret
		if rs, secret, err = PrepareRegisteredService(ctx, *serviceClass, mappings, data, remote_namespace); err != nil {
			errorList = append(errorList, err)
			continue
		}

		// modify the registered service
		if errs := handleFunc(ctx, remote_client, rs, secret); errs != nil {
			errorList = append(errorList, errs...)
		}
	}

	return errors.Join(errorList...)
}

func LookupServiceEndpointDescriptor(ctx context.Context, mappings []sed.SEDMapping, service unstructured.Unstructured) ([]v1alpha1.ServiceEndpointDefinitionItem, *v1.Secret, error) {
	var sedMappings []v1alpha1.ServiceEndpointDefinitionItem
	var errorList []error
	secret := &v1.Secret{StringData: map[string]string{}}
	secret.SetName(fmt.Sprintf("%s-descriptor", service.GetName()))
	for _, mapping := range mappings {
		value, err := mapping.ReadKey(ctx)
		if err != nil {
			errorList = append(errorList, err)
			continue
		}

		item := v1alpha1.ServiceEndpointDefinitionItem{
			Name:  mapping.Key(),
			Value: *value,
		}
		if mapping.InSecret() {
			item = v1alpha1.ServiceEndpointDefinitionItem{
				Name: mapping.Key(),
				ValueFromSecret: &v1alpha1.ServiceEndpointDefinitionSecretRef{
					Name: secret.GetName(),
					Key:  mapping.Key(),
				},
			}
			secret.StringData[mapping.Key()] = *value
		}
		sedMappings = append(sedMappings, item)
	}

	if len(errorList) != 0 {
		return nil, nil, errors.Join(errorList...)
	}

	if len(secret.StringData) == 0 {
		secret = nil
	}
	return sedMappings, secret, nil
}

func PrepareRegisteredService(
	ctx context.Context,
	serviceClass v1alpha1.ServiceClass,
	mappings []sed.SEDMapping,
	data unstructured.Unstructured,
	remote_namespace string,
) (v1alpha1.RegisteredService, *v1.Secret, error) {
	l := log.FromContext(ctx)
	sedMappings, secret, err := LookupServiceEndpointDescriptor(ctx, mappings, data)
	if err != nil {
		l.Error(err, "Failed to lookup service endpoint descriptor values",
			"name", data.GetName(),
			"namespace", data.GetNamespace(),
			"gvk", data.GroupVersionKind())
		return v1alpha1.RegisteredService{}, nil, err
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

	rs.Spec.Constraints = &v1alpha1.RegisteredServiceConstraints{
		Environments: serviceClass.Spec.GetEnvironmentConstraints(),
	}

	if secret != nil {
		secret.SetNamespace(remote_namespace)
	}
	return rs, secret, nil
}

func ServiceEndpointDefinitionMapping(cli client.Client, obj unstructured.Unstructured, serviceClass v1alpha1.ServiceClass) ([]sed.SEDMapping, error) {
	mappings := []sed.SEDMapping{}

	for _, mapping := range serviceClass.Spec.Resource.ServiceEndpointDefinitionMappings.ResourceFields {
		m, err := sed.NewSEDResourceMapping(obj, mapping)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}

	for _, m := range serviceClass.Spec.Resource.ServiceEndpointDefinitionMappings.SecretRefFields {
		m, err := sed.NewSEDSecretRefMapping(serviceClass.GetNamespace(), obj, cli, m)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}

	return mappings, nil
}

func (r *ServiceClassReconciler) setOwnerReference(ctx context.Context, scclass *v1alpha1.ServiceClass, owner metav1.Object) error {
	reconcileLog := log.FromContext(ctx)
	if err := ctrl.SetControllerReference(owner, scclass, r.Client.Scheme()); err != nil {
		return err
	}
	reconcileLog.Info("updating service class with owner reference", "service class", scclass.Spec)
	if err := r.Update(ctx, scclass); err != nil {
		return err
	}

	return nil
}

func (r *ServiceClassReconciler) SetWatchersForResources(ctx context.Context, serviceClass v1alpha1.ServiceClass) error {
	reconcileLog := log.FromContext(ctx)
	typemeta := metav1.TypeMeta{
		Kind:       serviceClass.Spec.Resource.Kind,
		APIVersion: serviceClass.Spec.Resource.APIVersion,
	}
	gvk := typemeta.GroupVersionKind()
	mapping, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		reconcileLog.Error(err, "error on creating mapping")
		return err
	}
	reconcileLog.Info("resource to be watched", "resource", mapping.Resource)
	if err = r.RunInformer(ctx, mapping.Resource, serviceClass); err != nil {
		reconcileLog.Error(err, "error running informer")
		return err
	}
	return nil
}

func (r *ServiceClassReconciler) RunInformer(ctx context.Context, resource schema.GroupVersionResource, serviceClass v1alpha1.ServiceClass) error {
	l := log.FromContext(ctx)

	// check if informer already exists
	if _, ok := r.informers[serviceClass.GetName()]; ok {
		l.Info("Informer already exists")
		return nil
	}
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		l.Info("failed creating cluster config")
		panic(err)
	}
	clusterClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		l.Info("failed creating cluster config")
		panic(err)
	}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, serviceClass.Namespace, nil)
	i := factory.ForResource(resource).Informer()

	var synced atomic.Bool
	synced.Store(false)
	if _, err := i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !synced.Load() {
				return
			}
			serviceClassResource := obj.(*unstructured.Unstructured)
			if err := r.CreateOrUpdateRegisteredService(ctx, *serviceClassResource, serviceClass); err != nil {
				return
			}
		},
		UpdateFunc: func(past, future interface{}) {
			if !synced.Load() {
				return
			}
			serviceClassResource := future.(*unstructured.Unstructured)
			if err := r.CreateOrUpdateRegisteredService(ctx, *serviceClassResource, serviceClass); err != nil {
				return
			}
		},
		DeleteFunc: func(obj interface{}) {
			if !synced.Load() {
				return
			}
			if err := r.DeleteRegisteredService(ctx, serviceClass); err != nil {
				return
			}
		},
	}); err != nil {
		return err
	}

	l.Info("run informer", "GroupVersionResource", resource)
	c, fc := context.WithCancel(ctx)

	li := informer{informer: i, ctx: c, cancelFunc: fc}
	r.informers[serviceClass.GetName()] = li
	go li.run()

	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		fc()
		delete(r.informers, serviceClass.GetName())
		return fmt.Errorf("could not sync cache")
	}

	synced.Store(true)

	return nil
}

func (r *ServiceClassReconciler) CreateOrUpdateRegisteredService(ctx context.Context, obj unstructured.Unstructured, serviceClass v1alpha1.ServiceClass) error {
	l := log.FromContext(ctx)
	var mappings []sed.SEDMapping
	var err error

	if mappings, err = ServiceEndpointDefinitionMapping(r.Client, obj, serviceClass); err != nil {
		return err
	}
	config, remote_namespace, err := workercluster.GetPrimazaKubeconfig(ctx)
	if err != nil {
		return err
	}
	remote_client, err := client.New(config, client.Options{
		Scheme: r.Client.Scheme(),
		Mapper: r.Client.RESTMapper(),
	})
	if err != nil {
		return err
	}
	errs := []error{err}
	var rs v1alpha1.RegisteredService
	var secret *v1.Secret
	if rs, secret, err = PrepareRegisteredService(ctx, serviceClass, mappings, obj, remote_namespace); err != nil {
		return err
	}
	spec := rs.Spec
	op, err := controllerutil.CreateOrUpdate(ctx, remote_client, &rs, func() error {
		rs.Spec = spec
		return nil
	})
	if err != nil {
		l.Error(err, "Failed to create or update registered service")
		errs = append(errs, err)
	} else {
		l.Info("Wrote registered service", "registered service", rs.Name, "namespace", rs.Namespace, "operation", op)
	}
	if secret != nil {
		data := secret.StringData
		_, err := controllerutil.CreateOrUpdate(ctx, remote_client, secret, func() error {
			secret.StringData = data
			return controllerutil.SetOwnerReference(&rs, secret, remote_client.Scheme())
		})
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (r *ServiceClassReconciler) DeleteRegisteredService(ctx context.Context, serviceClass v1alpha1.ServiceClass) error {
	l := log.FromContext(ctx)
	config, _, err := workercluster.GetPrimazaKubeconfig(ctx)
	if err != nil {
		return err
	}
	remote_client, err := client.New(config, client.Options{
		Scheme: r.Client.Scheme(),
		Mapper: r.Client.RESTMapper(),
	})
	if err != nil {
		return err
	}
	l.Info("remote cluster", "address", config.Host)

	// TODO: Deletion of Registered Services to be made dynamic
	registeredService := v1alpha1.RegisteredService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceClass.Name,
			Namespace: "primaza-system",
		},
	}
	if err = remote_client.Delete(ctx, &registeredService, &client.DeleteOptions{}); !apierrors.IsNotFound(err) {
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
