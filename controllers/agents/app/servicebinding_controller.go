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

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/dynamic"
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
	l := log.FromContext(ctx).WithValues("service binding", req.Name)
	l.Info("Reconciling service binding in agent app", "namespace", req.Namespace)

	var serviceBinding primazaiov1alpha1.ServiceBinding
	l.Info("retrieving ServiceBinding object", "ServiceBinding", serviceBinding)
	if err := r.Get(ctx, req.NamespacedName, &serviceBinding); err != nil {
		l.Error(err, "unable to retrieve ServiceBinding")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Check If service binding is deleted")
	if serviceBinding.HasDeletionTimestamp() {
		if controllerutil.ContainsFinalizer(&serviceBinding, ServiceBindingFinalizer) {
			if err := r.finalizeServiceBinding(ctx, serviceBinding); err != nil {
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
	}

	if err := r.ensureInformerIsRunningForServiceBinding(ctx, serviceBinding); err != nil {
		l.Error(err, "Failed to set watchers on ServiceBinding resources ", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, err
	}

	applications, err := r.getApplication(ctx, serviceBinding)
	if err != nil {
		// error retrieving the application(s), so setting the service binding status to false and reconcile
		s := primazaiov1alpha1.ServiceBindingStateReady
		c := metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.ServiceBindingBoundCondition,
			Status:             metav1.ConditionFalse,
			Reason:             conditionGetAppsFailureReason,
			Message:            err.Error(),
		}
		if err := r.updateServiceBindingStatus(ctx, &serviceBinding, c, s); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// retrieve ServiceBinding's Secret
	psSecret, err := r.GetSecret(ctx, serviceBinding, applications...)
	if err != nil {
		return ctrl.Result{}, err
	}

	// bind applications
	if err := r.PrepareBinding(ctx, &serviceBinding, psSecret, applications...); err != nil {
		return ctrl.Result{}, err
	}

	// set ServiceBinding Ownership on ServiceBinding's secret
	if err := ctrl.SetControllerReference(&serviceBinding, psSecret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Update(ctx, psSecret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) finalizeServiceBinding(ctx context.Context, serviceBinding primazaiov1alpha1.ServiceBinding) error {
	// need to stop the informers if the service class is deleted
	if i, ok := r.informers[serviceBinding.Name]; ok {
		i.cancelFunc()
		delete(r.informers, serviceBinding.Name)
	}

	applications, errs := r.getBoundApplications(ctx, serviceBinding)
	if err := r.unbindApplications(ctx, serviceBinding, applications...); err != nil {
		return err
	}
	return errors.Join(errs...)
}

func (r *ServiceBindingReconciler) GetSecret(ctx context.Context, serviceBinding primazaiov1alpha1.ServiceBinding, applications ...unstructured.Unstructured) (*v1.Secret, error) {
	psSecret := &v1.Secret{}
	secretLookupKey := client.ObjectKey{Name: serviceBinding.Spec.ServiceEndpointDefinitionSecret, Namespace: serviceBinding.Namespace}
	if secErr := r.Get(ctx, secretLookupKey, psSecret); secErr != nil {
		s := primazaiov1alpha1.ServiceBindingStateMalformed
		c := metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.ServiceBindingBoundCondition,
			Status:             metav1.ConditionFalse,
			Reason:             conditionGetSecretFailureReason,
			Message:            secErr.Error(),
		}
		if err := r.updateServiceBindingStatus(ctx, &serviceBinding, c, s); err != nil {
			return nil, err
		}

		if err := r.unbindApplications(ctx, serviceBinding, applications...); err != nil {
			return nil, err
		}
		return nil, secErr
	}

	return psSecret, nil
}

func (r *ServiceBindingReconciler) PrepareBinding(
	ctx context.Context,
	serviceBinding *primazaiov1alpha1.ServiceBinding,
	psSecret *v1.Secret,
	applications ...unstructured.Unstructured,
) error {
	l := log.FromContext(ctx)

	f := false
	p := int32(0444)
	volumeName := serviceBinding.Name
	mountPathDir := serviceBinding.Name

	volumeProjection := &v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  serviceBinding.Spec.ServiceEndpointDefinitionSecret,
				Optional:    &f,
				DefaultMode: &p,
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

func (r *ServiceBindingReconciler) prepareContainerWithMounts(
	ctx context.Context,
	sb *primazaiov1alpha1.ServiceBinding,
	psSecret *v1.Secret,
	mountPathDir, volumeName string,
	unstructuredVolume map[string]interface{},
	application unstructured.Unstructured,
) error {
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
		if err = r.updateContainerInfo(ctx, containers, *sb, mountPathDir, volumeName, psSecret); err != nil {
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

func (r *ServiceBindingReconciler) bindApplications(
	ctx context.Context,
	sb *primazaiov1alpha1.ServiceBinding,
	psSecret *v1.Secret,
	mountPathDir,
	volumeName string,
	unstructuredVolume map[string]interface{},
	applications ...unstructured.Unstructured,
) error {
	sb.Status.Connections = []primazaiov1alpha1.BoundWorkload{}

	var el []error
	for _, application := range applications {
		err := r.prepareContainerWithMounts(ctx, sb, psSecret, mountPathDir, volumeName, unstructuredVolume, application)
		if err != nil {
			el = append(el, err)
		}

		b := primazaiov1alpha1.BoundWorkload{
			Name: application.GetName(),
		}
		sb.Status.Connections = append(sb.Status.Connections, b)
	}

	if err := r.updateServiceBindingStatusWithBindingResult(ctx, sb, el); err != nil {
		return err
	}
	return errors.Join(el...)
}

func (r *ServiceBindingReconciler) updateServiceBindingStatusWithBindingResult(
	ctx context.Context,
	sb *primazaiov1alpha1.ServiceBinding,
	bindingErrors []error,
) error {
	c, s := func() (metav1.Condition, primazaiov1alpha1.ServiceBindingState) {
		if len(bindingErrors) != 0 {
			cerr := errors.Join(bindingErrors...)
			return metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               primazaiov1alpha1.ServiceBindingBoundCondition,
				Status:             metav1.ConditionFalse,
				Reason:             conditionBindingFailure,
				Message:            cerr.Error(),
			}, primazaiov1alpha1.ServiceBindingStateMalformed
		}

		return metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.ServiceBindingBoundCondition,
			Status:             metav1.ConditionTrue,
			Reason:             conditionBindingSuccessful,
			Message:            "",
		}, primazaiov1alpha1.ServiceBindingStateReady
	}()

	return r.updateServiceBindingStatus(ctx, sb, c, s)
}

func (r *ServiceBindingReconciler) updateServiceBindingStatus(
	ctx context.Context,
	sb *primazaiov1alpha1.ServiceBinding,
	condition metav1.Condition,
	state primazaiov1alpha1.ServiceBindingState,
) error {
	l := log.FromContext(ctx).WithValues("service-binding", sb.Name)

	l.Info("set the status of the service binding")
	meta.SetStatusCondition(&sb.Status.Conditions, condition)
	sb.Status.State = state

	l.Info("updating the service binding status")
	if err := r.Status().Update(ctx, sb); err != nil {
		l.Error(err, "unable to update the service binding")
		return err
	}
	l.Info("service binding status updated")

	return nil
}

func (r *ServiceBindingReconciler) getBoundApplications(
	ctx context.Context,
	sb primazaiov1alpha1.ServiceBinding,
) ([]unstructured.Unstructured, []error) {
	var applications []unstructured.Unstructured
	l := log.FromContext(ctx).WithValues("service-binding", sb.Name)

	lookupWorkload := func(name string) (*unstructured.Unstructured, error) {
		applicationLookupKey := client.ObjectKey{Name: name, Namespace: sb.Namespace}

		application := unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       sb.Spec.Application.Kind,
				"apiVersion": sb.Spec.Application.APIVersion,
			},
		}

		l = l.WithValues(
			"application Kind", sb.Spec.Application.Kind,
			"application APIVersion", sb.Spec.Application.APIVersion,
			"application Name", name,
		)

		if err := r.Get(ctx, applicationLookupKey, &application); err != nil {
			l.Error(err, "unable to retrieve Application")
			return nil, err
		}
		l.Info("application object retrieved")
		return &application, nil
	}

	errs := []error{}
	for _, bw := range sb.Status.Connections {
		if w, err := lookupWorkload(bw.Name); err != nil {
			errs = append(errs, err)
		} else {
			applications = append(applications, *w)
		}
	}
	return applications, errs
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
	} else if sb.Spec.Application.Selector != nil {
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

func (r *ServiceBindingReconciler) updateContainerInfo(
	ctx context.Context,
	containers []interface{},
	sb primazaiov1alpha1.ServiceBinding,
	mountPathDir, volumeName string,
	psSecret *v1.Secret,
) error {
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

func verifyApplicationSatisfiesServiceBindingSpec(obj *unstructured.Unstructured, sb primazaiov1alpha1.ServiceBinding) bool {
	switch {
	case sb.Spec.Application.Name == obj.GetName():
		return true
	case sb.Spec.Application.Selector != nil:
		// TODO: handle sb.Spec.Application.Selector.ByLabels.MatchExpressions
		for label, value := range obj.GetLabels() {
			if val, ok := sb.Spec.Application.Selector.MatchLabels[label]; ok && value == val {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ServiceBinding{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
