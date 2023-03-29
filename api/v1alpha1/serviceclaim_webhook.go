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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceclaimlog = logf.Log.WithName("serviceclaim-resource")

type serviceClaimValidator struct {
	client client.Client
}

func (r *ServiceClaim) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&serviceClaimValidator{
			client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-primaza-io-v1alpha1-serviceclaim,mutating=false,failurePolicy=fail,sideEffects=None,groups=primaza.io,resources=serviceclaims,verbs=create;update,versions=v1alpha1,name=vserviceclaim.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &serviceClaimValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *serviceClaimValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return err
	}

	serviceclaimlog.Info("validate create", "name", r.Name)
	return v.validate(r)
}

func (v *serviceClaimValidator) validate(r *ServiceClaim) error {
	if r.Spec.ApplicationClusterContext != nil && r.Spec.EnvironmentTag != "" {
		return fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be used together")
	}
	if r.Spec.ApplicationClusterContext == nil && r.Spec.EnvironmentTag == "" {
		return fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be empty")
	}
	return nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *serviceClaimValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) error {
	r, ok := newObj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", newObj.GetObjectKind().GroupVersionKind())
		return err
	}

	serviceclaimlog.Info("validate update", "name", r.Name)
	return v.validate(r)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *serviceClaimValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return err
	}

	serviceclaimlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
