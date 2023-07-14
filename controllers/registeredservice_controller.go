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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/envtag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (r *RegisteredServiceReconciler) registerHealthcheck(ctx context.Context, rs *primazaiov1alpha1.RegisteredService) error {
	// Make a CronJob that runs the healthcheck.  The CronJob handler will take
	// care of the rest.
	cronjob := batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rs.Name,
			Namespace: rs.Namespace,
		},
	}
	operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &cronjob, func() error {
		var one int32 = 1
		var two int32 = 2
		var thirty int64 = 30
		var uid int64 = 65530
		t := true
		f := false
		container := corev1.Container{
			Name:            "healthcheck",
			Command:         rs.Spec.HealthCheck.Container.Command,
			Image:           rs.Spec.HealthCheck.Container.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				Privileged:               &f,
				RunAsNonRoot:             &t,
				AllowPrivilegeEscalation: &f,
			},
		}
		cronjob.Spec = batchv1.CronJobSpec{
			// run every n minutes
			Schedule:                   fmt.Sprintf("*/%d * * * *", rs.Spec.HealthCheck.Container.Minutes),
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &one,
			FailedJobsHistoryLimit:     &one,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					ActiveDeadlineSeconds: &thirty,
					BackoffLimit:          &two,
					Parallelism:           &one,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers:    []corev1.Container{container},
							RestartPolicy: corev1.RestartPolicyNever,
							SecurityContext: &corev1.PodSecurityContext{
								RunAsUser:    &uid,
								RunAsGroup:   &uid,
								RunAsNonRoot: &t,
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(rs, &cronjob, r.Scheme)
	})

	if err != nil {
		return err
	}

	l := log.FromContext(ctx).WithValues("cronjob", cronjob.Name, "namespace", cronjob.Namespace)
	if operation == controllerutil.OperationResultCreated {
		l.Info("Created healthcheck cronjob")
	} else {
		l.Info("Modified healthcheck cronjob", "operation", operation)
	}

	return nil
}

func (r *RegisteredServiceReconciler) handleHealthcheck(ctx context.Context, rs *primazaiov1alpha1.RegisteredService) error {
	l := log.FromContext(ctx).WithValues("namespace", rs.Namespace, "registered service", rs.Name)
	cronjob := batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: rs.Name, Namespace: rs.Namespace}, &cronjob)
	if k8errors.IsNotFound(err) {
		l.Info("creating registered service healthcheck")
		err := r.registerHealthcheck(ctx, rs)
		rs.Status.State = primazaiov1alpha1.RegisteredServiceStateUnknown
		return err
	} else if err != nil {
		return err
	}

	jobList := batchv1.JobList{}
	err = r.List(ctx, &jobList, client.InNamespace(cronjob.Namespace))
	if err != nil {
		return err
	}

	completed := false
	failed := false

	for _, job := range jobList.Items {
		found := false
		for _, ownerReference := range job.OwnerReferences {
			if ownerReference.UID == cronjob.UID {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				completed = true
				break
			} else if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				failed = true
				break
			}
		}
	}

	l.Info("Job status", "completed", completed, "failed", failed)
	if completed {
		// only keep it in 'Claimed' if the healthcheck succeeded
		if rs.Status.State != primazaiov1alpha1.RegisteredServiceStateClaimed {
			rs.Status.State = primazaiov1alpha1.RegisteredServiceStateAvailable
		}
	} else if failed {
		rs.Status.State = primazaiov1alpha1.RegisteredServiceStateUnreachable
	} else {
		rs.Status.State = primazaiov1alpha1.RegisteredServiceStateUnknown
	}

	return r.registerHealthcheck(ctx, rs)
}

func (r *RegisteredServiceReconciler) cleanupHealthchecks(ctx context.Context, rs *primazaiov1alpha1.RegisteredService) error {
	cronjobList := batchv1.CronJobList{}
	err := r.List(ctx, &cronjobList, client.InNamespace(rs.Namespace))
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	errs := []error{}
	for _, cronjob := range cronjobList.Items {
		// TODO: revisit this once the loopvar experiment becomes stable (go 1.21?)
		cronjob := cronjob

		owned := false
		for _, ownerRef := range cronjob.OwnerReferences {
			if ownerRef.UID == rs.UID {
				owned = true
				break
			}
		}

		if !owned {
			continue
		}

		err := r.Delete(ctx, &cronjob)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=registeredservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,namespace=system,resources=servicecatalogs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,namespace=system,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,namespace=system,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch,namespace=system,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,namespace=system,resources=jobs/status,verbs=get;update;patch

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
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)

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

	if rs.Spec.HealthCheck != nil {
		err := r.handleHealthcheck(ctx, &rs)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		err := r.cleanupHealthchecks(ctx, &rs)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Since we don't have a healthcheck, we can be in one of two states:
		// Available or Claimed.  Explicitly set to Available if not Claimed,
		// since this also lets us clean up healthcheck removal.
		if rs.Status.State != primazaiov1alpha1.RegisteredServiceStateClaimed {
			rs.Status.State = primazaiov1alpha1.RegisteredServiceStateAvailable
		}
	}

	if rs.Status.State == primazaiov1alpha1.RegisteredServiceStateAvailable {
		err = r.reconcileCatalogs(ctx, rs)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error adding service to ServiceCatalog")
			return ctrl.Result{}, err
		}
	} else {
		err = r.removeServiceFromCatalogs(ctx, req.Namespace, req.Name)

		if err != nil {
			// Service Catalog update failed
			log.Error(err, "Error removing service from ServiceCatalog")
			return ctrl.Result{}, err
		}
	}

	log.Info("Updating status of RegisteredService", "state", rs.Status.State)
	err = r.Status().Update(ctx, &rs)
	if err != nil {
		log.Error(err, "RegisteredService Status Failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegisteredServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.RegisteredService{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}
