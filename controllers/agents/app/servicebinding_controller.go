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
	"path"
	"strings"
	"time"

	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"go.uber.org/atomic"
	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	conditionGetAppsFailureReason   = "NoMatchingWorkloads"
	conditionGetSecretFailureReason = "ErrorFetchSecret"
	conditionBindingSuccessful      = "Successful"
	conditionBindingFailure         = "Binding Failure"
)

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

// ServiceBindingRoot points to the environment variable in the container
// which is used as the volume mount path.  In the absence of this
// environment variable, `/bindings` is used as the volume mount path.
// Refer: https://github.com/servicebinding/spec#reconciler-implementation
const (
	ServiceBindingRoot      = "SERVICE_BINDING_ROOT"
	ServiceBindingFinalizer = "servicebindings.primaza.io/finalizer"
)

func NewServiceBindingReconciler(mgr ctrl.Manager) *ServiceBindingReconciler {
	return &ServiceBindingReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Interface: dynamic.NewForConfigOrDie(mgr.GetConfig()),
		informers: make(map[string]informer, 0),
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ServiceBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling service binding in agent app", "namespace", req.Namespace, "name", req.Name)

	l.Info("starting reconciliation")

	var serviceBinding primazaiov1alpha1.ServiceBinding

	l.Info("retrieving ServiceBinding object", "ServiceBinding", serviceBinding)
	if err := r.Get(ctx, req.NamespacedName, &serviceBinding); err != nil {
		l.Error(err, "unable to retrieve ServiceBinding")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	l.Info("ServiceBinding object retrieved", "ServiceBinding", serviceBinding)

	applications, err := r.getApplication(ctx, serviceBinding)
	if err != nil {
		// error retrieving the application(s), so setting the service binding status to false and reconcile
		if errUpdateStatus := r.setStatus(ctx, serviceBinding, metav1.ConditionFalse, conditionGetAppsFailureReason, primazaiov1alpha1.ServiceBindingStateReady, err.Error(), primazaiov1alpha1.ServiceBindingBoundCondition); errUpdateStatus != nil {
			return ctrl.Result{}, errUpdateStatus
		}
		return ctrl.Result{}, err
	}
	l.Info("Check If service binding is deleted")
	if serviceBinding.HasDeletionTimestamp() {
		if controllerutil.ContainsFinalizer(&serviceBinding, ServiceBindingFinalizer) {

			if err = r.finalizeServiceBinding(ctx, serviceBinding, applications); err != nil {
				l.Error(err, "Error on unbinding applications on Service Binding Deletion")
				return ctrl.Result{}, err
			}

			// Remove finalizer from service binding
			if finalizerBool := controllerutil.RemoveFinalizer(&serviceBinding, ServiceBindingFinalizer); !finalizerBool {
				l.Error(errors.New("Finalizer not removed for service binding"), "Finalizer not removed for service binding")
				return ctrl.Result{}, errors.New("Finalizer not removed for service binding")
			}
			if err := r.Update(ctx, &serviceBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	l.Info("Add Finalizer if needed")
	// add finalizer if needed
	if !controllerutil.ContainsFinalizer(&serviceBinding, ServiceBindingFinalizer) {
		controllerutil.AddFinalizer(&serviceBinding, ServiceBindingFinalizer)
		if err := r.Update(ctx, &serviceBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.SetWatchersForResources(ctx, serviceBinding); err != nil {
		l.Error(err, "Failed to set watchers on ServiceBinding resources ", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	var psSecret *v1.Secret
	if psSecret, err = r.GetSecret(ctx, serviceBinding, applications); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.PrepareBinding(ctx, serviceBinding, applications, psSecret); err != nil {
		return ctrl.Result{}, err
	}

	if err := ctrl.SetControllerReference(&serviceBinding, psSecret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.Update(ctx, psSecret)
}

func (r *ServiceBindingReconciler) finalizeServiceBinding(ctx context.Context, serviceBinding v1alpha1.ServiceBinding, applications []unstructured.Unstructured) error {
	// need to stop the informers if the service class is deleted
	if i, ok := r.informers[serviceBinding.Name]; ok {
		i.cancelFunc()
		delete(r.informers, serviceBinding.Name)
	}
	err := r.unbindApplications(ctx, serviceBinding, applications...)
	if err != nil {
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) GetSecret(ctx context.Context, serviceBinding v1alpha1.ServiceBinding, applications []unstructured.Unstructured) (*v1.Secret, error) {
	psSecret := &v1.Secret{}
	secretLookupKey := client.ObjectKey{Name: serviceBinding.Spec.ServiceEndpointDefinitionSecret, Namespace: serviceBinding.Namespace}
	if secErr := r.Get(ctx, secretLookupKey, psSecret); secErr != nil {
		// error retrieving the application(s), so setting the service binding status to false and reconcile
		err := r.setStatus(ctx, serviceBinding, metav1.ConditionFalse, conditionGetSecretFailureReason, primazaiov1alpha1.ServiceBindingStateMalformed, secErr.Error(), primazaiov1alpha1.ServiceBindingNotBoundCondition)
		if err != nil {
			return nil, err
		}
		err = r.unbindApplications(ctx, serviceBinding, applications...)
		if err != nil {
			return nil, err
		}
		return nil, secErr
	}
	return psSecret, nil
}

func (r *ServiceBindingReconciler) PrepareBinding(ctx context.Context, serviceBinding v1alpha1.ServiceBinding, applications []unstructured.Unstructured, psSecret *v1.Secret) error {
	l := log.FromContext(ctx)

	volumeName := serviceBinding.Name
	mountPathDir := serviceBinding.Name
	sp := &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: serviceBinding.Spec.ServiceEndpointDefinitionSecret,
		}}

	volumeProjection := &v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources: []v1.VolumeProjection{{Secret: sp}},
			},
		},
	}
	l.Info("converting the volumeProjection to an unstructured object", "Volume", volumeProjection)
	unstructuredVolume, err := runtime.DefaultUnstructuredConverter.ToUnstructured(volumeProjection)
	if err != nil {
		l.Error(err, "unable to convert volumeProjection to an unstructured object")
		return err
	}

	err = r.bindApplications(ctx, serviceBinding, psSecret, mountPathDir, volumeName, unstructuredVolume, applications...)
	if err != nil {
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) prepareContainerWithMounts(ctx context.Context,
	sb primazaiov1alpha1.ServiceBinding, psSecret *v1.Secret, mountPathDir, volumeName string, unstructuredVolume map[string]interface{}, application unstructured.Unstructured) error {
	l := log.FromContext(ctx)
	l.Info("Prepare application mounting")

	containersPaths := [][]string{}
	volumesPath := []string{"spec", "template", "spec", "volumes"}

	containersPaths = append(containersPaths,
		[]string{"spec", "template", "spec", "containers"},
		[]string{"spec", "template", "spec", "initContainers"},
	)
	l.Info("referencing the volume in an unstructured object")
	volumes, found, err := unstructured.NestedSlice(application.Object, volumesPath...)
	if err != nil {
		l.Error(err, "unable to reference the volumes in the application object")
		return err
	}
	if !found {
		l.Info("volumes not found in the application object")
	}
	l.Info("Volumes values", "volumes", volumes)

	volumeFound := false

	for i, volume := range volumes {
		l.Info("Volume", "volume", volume)
		if volume.(map[string]interface{})["name"].(string) == volumeName {
			volumes[i] = unstructuredVolume
			volumeFound = true
			break
		}
	}

	if !volumeFound {
		volumes = append(volumes, unstructuredVolume)
	}
	l.Info("setting the updated volumes into the application using the unstructured object")
	if err := unstructured.SetNestedSlice(application.Object, volumes, volumesPath...); err != nil {
		return err
	}
	l.Info("application object after setting the update volume", "Application", application)

	for _, containersPath := range containersPaths {
		l.Info("referencing containers in an unstructured object")
		containers, found, err := unstructured.NestedSlice(application.Object, containersPath...)
		if err != nil {
			l.Error(err, "unable to reference containers in the application object")
			return err
		}
		if !found {
			e := &field.Error{Type: field.ErrorTypeRequired, Field: strings.Join(containersPath, "."), Detail: "no containers"}
			l.Info("containers not found in the application object", "error", e)
		}

		l.Info("update container with volume and volume mounts", "containers", containers)
		if err = r.updateContainerInfo(ctx, containers, sb, mountPathDir, volumeName, psSecret); err != nil {
			return err
		}

		l.Info("setting the updated containers into the application using the unstructured object")
		if err := unstructured.SetNestedSlice(application.Object, containers, containersPath...); err != nil {
			return err
		}
		l.Info("application object after setting the updated containers", "Application", application)
	}

	l.Info("updating the application with updated volumes and volumeMounts")
	if err := r.Update(ctx, &application); err != nil {
		l.Error(err, "unable to update the application", "application", application)
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) bindApplications(ctx context.Context,
	sb primazaiov1alpha1.ServiceBinding, psSecret *v1.Secret, mountPathDir, volumeName string, unstructuredVolume map[string]interface{}, applications ...unstructured.Unstructured) error {

	l := log.FromContext(ctx)

	var el []error
	for _, application := range applications {
		err := r.prepareContainerWithMounts(ctx, sb, psSecret, mountPathDir, volumeName, unstructuredVolume, application)
		if err != nil {
			el = append(el, err)
		}
	}
	l.Info("set the status of the service binding")
	if len(el) != 0 {
		cerr := errors.Join(el...)
		err := r.setStatus(ctx, sb, metav1.ConditionFalse, conditionBindingFailure, primazaiov1alpha1.ServiceBindingStateMalformed, cerr.Error(), primazaiov1alpha1.ServiceBindingNotBoundCondition)
		if err != nil {
			return err
		}
		return cerr
	}
	err := r.setStatus(ctx, sb, metav1.ConditionTrue, conditionBindingSuccessful, primazaiov1alpha1.ServiceBindingStateReady, "", primazaiov1alpha1.ServiceBindingBoundCondition)
	if err != nil {
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) setStatus(ctx context.Context,
	sb primazaiov1alpha1.ServiceBinding, conditionStatus metav1.ConditionStatus, reason, state, message, conditionType string) error {
	l := log.FromContext(ctx)
	c := metav1.Condition{
		LastTransitionTime: metav1.NewTime(time.Now()),
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
	}
	meta.SetStatusCondition(&sb.Status.Conditions, c)
	sb.Status.State = state

	l.Info("updating the service binding status")
	if err := r.Status().Update(ctx, &sb); err != nil {
		l.Error(err, "unable to update the service binding", "ServiceBinding", sb)
		return err
	}
	l.Info("service binding status updated", "ServiceBinding", sb)

	return nil
}

func (r *ServiceBindingReconciler) getApplication(ctx context.Context,
	sb primazaiov1alpha1.ServiceBinding) ([]unstructured.Unstructured, error) {
	var applications []unstructured.Unstructured
	l := log.FromContext(ctx)
	if sb.Spec.Application.Name != "" {
		applicationLookupKey := client.ObjectKey{Name: sb.Spec.Application.Name, Namespace: sb.Namespace}

		application := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       sb.Spec.Application.Kind,
				"apiVersion": sb.Spec.Application.APIVersion,
				"metadata": map[string]interface{}{
					"name":      sb.Spec.Application.Name,
					"namespace": sb.Namespace,
				},
			},
		}

		l.Info("retrieving the application object", "Application", application)
		if err := r.Get(ctx, applicationLookupKey, application); err != nil {
			l.Error(err, "unable to retrieve Application")
			return []unstructured.Unstructured{}, err
		}
		l.Info("application object retrieved", "Application", application)
		applications = append(applications, *application)
	}

	if sb.Spec.Application.Selector != nil {
		applicationList := &unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"kind":       sb.Spec.Application.Kind,
				"apiVersion": sb.Spec.Application.APIVersion,
			},
		}

		l.Info("retrieving the application objects", "Application", applicationList)
		opts := &client.ListOptions{
			LabelSelector: labels.Set(sb.Spec.Application.Selector.MatchLabels).AsSelector(),
			Namespace:     sb.Namespace,
		}

		if err := r.List(ctx, applicationList, opts); err != nil {
			l.Error(err, "unable to retrieve Application using labels")
			return []unstructured.Unstructured{}, err
		}
		l.Info("application objects retrieved", "Application", applicationList)
		applications = append(applications, applicationList.Items...)
	}
	if len(applications) == 0 {
		// Requeue with a time interval is required as the applications is not available to reconcile
		// In future, probably watching for applications os specific types (Deployment, CronJob etc.) based
		// on label can be introduced or a webhook can detect application change and trigger reconciliation
		return nil, fmt.Errorf("applications not found")
	}
	return applications, nil
}

func removeServiceBindingEnvironments(envList []v1.EnvVar, sb primazaiov1alpha1.ServiceBinding) []v1.EnvVar {
	var envListCopy []v1.EnvVar
	for _, val := range envList {
		if val.ValueFrom != nil && val.ValueFrom.SecretKeyRef != nil &&
			val.ValueFrom.SecretKeyRef.Name != sb.Spec.ServiceEndpointDefinitionSecret {
			envListCopy = append(envListCopy, val)
		}
	}
	return envListCopy
}

func (r *ServiceBindingReconciler) updateContainerInfo(ctx context.Context, containers []interface{}, sb primazaiov1alpha1.ServiceBinding, mountPathDir, volumeName string, psSecret *v1.Secret) error {

	l := log.FromContext(ctx)
	for i := range containers {
		container := &containers[i]
		l.Info("updating container", "container", container)
		c := &v1.Container{}
		u := (*container).(map[string]interface{})
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, c); err != nil {
			return err
		}

		// first remove the present environment variables
		c.Env = removeServiceBindingEnvironments(c.Env, sb)
		// update environment variables
		for _, e := range sb.Spec.Envs {
			env := v1.EnvVar{
				Name: e.Name,
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						Key: e.Key,
						LocalObjectReference: v1.LocalObjectReference{
							Name: psSecret.Name,
						},
					},
				},
			}
			c.Env = append(c.Env, env)
		}

		mountPath := ""
		for _, e := range c.Env {
			if e.Name == ServiceBindingRoot {
				mountPath = path.Join(e.Value, mountPathDir)
				break
			}
		}

		if mountPath == "" {
			mountPath = path.Join("/bindings", mountPathDir)
			c.Env = append(c.Env, v1.EnvVar{
				Name:  ServiceBindingRoot,
				Value: "/bindings",
			})
		}

		volumeMount := v1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			ReadOnly:  true,
		}

		volumeMountFound := false
		for j, vm := range c.VolumeMounts {
			if vm.Name == volumeName {
				c.VolumeMounts[j] = volumeMount
				volumeMountFound = true
				break
			}
		}

		if !volumeMountFound {
			c.VolumeMounts = append(c.VolumeMounts, volumeMount)
		}

		nu, err := runtime.DefaultUnstructuredConverter.ToUnstructured(c)
		if err != nil {
			return err
		}

		containers[i] = nu
	}
	return nil

}

func (r *ServiceBindingReconciler) removeBindingInformationFromContainer(ctx context.Context, sb primazaiov1alpha1.ServiceBinding, containers []interface{}, volumeName, mountPathDir, secretName string) error {
	l := log.FromContext(ctx)
	for i := range containers {
		container := &containers[i]
		l.Info("updating container", "container", container)
		c := &v1.Container{}
		u := (*container).(map[string]interface{})
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, c); err != nil {
			return err
		}

		for i, volumeMount := range c.VolumeMounts {
			if volumeMount.Name == volumeName {
				c.VolumeMounts = append(c.VolumeMounts[:i], c.VolumeMounts[i+1:]...)
			}
		}

		for i, env := range c.Env {
			if env.Name == ServiceBindingRoot {
				c.Env = append(c.Env[:i], c.Env[(i+1):]...)
			}
		}
		c.Env = removeServiceBindingEnvironments(c.Env, sb)
		nu, err := runtime.DefaultUnstructuredConverter.ToUnstructured(c)
		if err != nil {
			return err
		}

		containers[i] = nu
	}
	return nil
}

func (r *ServiceBindingReconciler) removeVolumeMountAndEnvironment(ctx context.Context, sb primazaiov1alpha1.ServiceBinding, application unstructured.Unstructured, volumeName, mountPathDir, secretName string) error {
	l := log.FromContext(ctx)
	l.Info("Prepare removing application mounting")

	containersPaths := [][]string{}
	volumesPath := []string{"spec", "template", "spec", "volumes"}

	containersPaths = append(containersPaths,
		[]string{"spec", "template", "spec", "containers"},
		[]string{"spec", "template", "spec", "initContainers"},
	)
	l.Info("referencing the volume in an unstructured object")
	volumes, found, err := unstructured.NestedSlice(application.Object, volumesPath...)
	if err != nil {
		l.Error(err, "unable to reference the volumes in the application object")
		return err
	}
	// check if volume not found in application object
	if !found {
		l.Info("volumes not found in the application object")
		return nil
	}
	for i, volume := range volumes {
		l.Info("Volume", "volume", volume)
		if volume.(map[string]interface{})["name"].(string) == volumeName {
			volumes = append(volumes[:i], volumes[i+1:]...)
		}
	}

	l.Info("setting the updated volumes into the application using the unstructured object")
	if err := unstructured.SetNestedSlice(application.Object, volumes, volumesPath...); err != nil {
		return err
	}
	l.Info("application object after setting the update volume", "Application", application)

	for _, containersPath := range containersPaths {
		l.Info("referencing containers in an unstructured object")
		containers, found, err := unstructured.NestedSlice(application.Object, containersPath...)
		if err != nil {
			l.Error(err, "unable to reference containers in the application object")
			return err
		}
		if !found {
			e := &field.Error{Type: field.ErrorTypeRequired, Field: strings.Join(containersPath, "."), Detail: "no containers"}
			l.Info("containers not found in the application object", "error", e)
		}

		l.Info("remove volume mounts from containers", "containers", containers)
		if err = r.removeBindingInformationFromContainer(ctx, sb, containers, mountPathDir, volumeName, secretName); err != nil {
			return err
		}

		l.Info("setting the updated containers into the application using the unstructured object")
		if err := unstructured.SetNestedSlice(application.Object, containers, containersPath...); err != nil {
			return err
		}
		l.Info("application object after setting the updated containers", "Application", application)
	}

	l.Info("updating the application with updated volumes and volumeMounts")
	if err := r.Update(ctx, &application); err != nil {
		l.Error(err, "unable to update the application", "application", application)
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) unbindApplications(ctx context.Context,
	serviceBinding primazaiov1alpha1.ServiceBinding, applications ...unstructured.Unstructured) error {
	var el []error
	volumeName := serviceBinding.Name
	mountPathDir := serviceBinding.Name
	secretName := serviceBinding.Spec.ServiceEndpointDefinitionSecret
	for _, application := range applications {
		err := r.removeVolumeMountAndEnvironment(ctx, serviceBinding, application, volumeName, mountPathDir, secretName)
		if err != nil {
			el = append(el, err)
		}
	}
	if len(el) != 0 {
		cerr := errors.Join(el...)
		return cerr
	}
	return nil
}

func (r *ServiceBindingReconciler) SetWatchersForResources(ctx context.Context, serviceBinding v1alpha1.ServiceBinding) error {
	reconcileLog := log.FromContext(ctx)
	typemeta := metav1.TypeMeta{
		Kind:       serviceBinding.Spec.Application.Kind,
		APIVersion: serviceBinding.Spec.Application.APIVersion,
	}
	gvk := typemeta.GroupVersionKind()
	mapping, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		reconcileLog.Error(err, "error on creating mapping")
		return err
	}
	reconcileLog.Info("resource to be watched", "resource", mapping.Resource)
	if err = r.RunInformer(ctx, mapping.Resource, serviceBinding); err != nil {
		reconcileLog.Error(err, "error running informer")
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) RunInformer(ctx context.Context, resource schema.GroupVersionResource, serviceBinding v1alpha1.ServiceBinding) error {
	l := log.FromContext(ctx)

	// check if informer already exists
	if _, ok := r.informers[serviceBinding.GetName()]; ok {
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
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, serviceBinding.Namespace, nil)
	i := factory.ForResource(resource).Informer()

	var synced atomic.Bool
	synced.Store(false)
	if _, err := i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !synced.Load() {
				return
			}
			var psSecret *v1.Secret
			var applicationResourceList []unstructured.Unstructured
			applicationResource := obj.(*unstructured.Unstructured)
			if !verifyApplicationSatisfiesServiceBindingSpec(applicationResource, serviceBinding) {
				return
			}
			applicationResourceList = append(applicationResourceList, *applicationResource)
			l.Info(fmt.Sprintf("application resource %v", applicationResourceList))
			if psSecret, err = r.GetSecret(ctx, serviceBinding, applicationResourceList); err != nil {
				l.Error(err, "Informer AddEventHandler: Error retrieving secret")
				return
			}
			if err := r.PrepareBinding(ctx, serviceBinding, applicationResourceList, psSecret); err != nil {
				l.Error(err, "Informer AddEventHandler: Error preparing binding")
				return
			}
		},
		UpdateFunc: func(past, future interface{}) {
			if !synced.Load() {
				return
			}
			var applicationResourceList []unstructured.Unstructured
			applicationResource := future.(*unstructured.Unstructured)
			applicationResourceList = append(applicationResourceList, *applicationResource)
			l.Info(fmt.Sprintf("application resource %v", applicationResourceList))
			if !verifyApplicationSatisfiesServiceBindingSpec(applicationResource, serviceBinding) {
				return
			}
			var psSecret *v1.Secret
			if psSecret, err = r.GetSecret(ctx, serviceBinding, applicationResourceList); err != nil {
				l.Error(err, "Informer AddEventHandler: Error retrieving secret")
				return
			}
			if err := r.PrepareBinding(ctx, serviceBinding, applicationResourceList, psSecret); err != nil {
				l.Error(err, "Informer AddEventHandler: Error preparing binding")
				return
			}
		},
		DeleteFunc: func(obj interface{}) {
			if !synced.Load() {
				return
			}
			applicationResource := obj.(*unstructured.Unstructured)
			applications, _ := r.getApplication(ctx, serviceBinding)
			if !verifyApplicationSatisfiesServiceBindingSpec(applicationResource, serviceBinding) {
				return
			}

			if len(applications) != 0 {
				return
			}
			var sb primazaiov1alpha1.ServiceBinding
			if err := r.Get(ctx, types.NamespacedName{Name: serviceBinding.Name, Namespace: serviceBinding.Namespace}, &sb); err == nil {
				// applications are deleted, so setting the service binding status to false and reconcile
				if errUpdateStatus := r.setStatus(ctx,
					sb,
					metav1.ConditionFalse,
					conditionGetAppsFailureReason,
					primazaiov1alpha1.ServiceBindingStateReady,
					"application was bound with the secret but the application got deleted",
					primazaiov1alpha1.ServiceBindingBoundCondition); errUpdateStatus != nil {
					l.Error(errUpdateStatus, "error on updating status")
					return
				}
			}
		},
	}); err != nil {
		return err
	}

	l.Info("run informer", "GroupVersionResource", resource)
	c, fc := context.WithCancel(ctx)

	li := informer{informer: i, ctx: c, cancelFunc: fc}
	r.informers[serviceBinding.GetName()] = li
	go li.run()

	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		fc()
		delete(r.informers, serviceBinding.GetName())
		return fmt.Errorf("could not sync cache")
	}

	synced.Store(true)

	return nil
}

func verifyApplicationSatisfiesServiceBindingSpec(obj *unstructured.Unstructured, sb v1alpha1.ServiceBinding) bool {
	switch {
	case sb.Spec.Application.Name == obj.GetName():
		return true
	case sb.Spec.Application.Selector != nil:
		for label, value := range obj.GetLabels() {
			val, ok := sb.Spec.Application.Selector.MatchLabels[label]
			return ok && value == val
		}
	default:
		return false
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceBinding{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
