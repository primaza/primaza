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
	"reflect"

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
func (v *serviceClaimValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclaimlog.Info("validate create", "name", r.Name)
	return v.validate(r)
}

func (v *serviceClaimValidator) validate(r *ServiceClaim) (admission.Warnings, error) {
	if r.Spec.ApplicationClusterContext != nil && r.Spec.EnvironmentTag != "" {
		return nil, fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be used together")
	}
	if r.Spec.ApplicationClusterContext == nil && r.Spec.EnvironmentTag == "" {
		return nil, fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be empty")
	}
	if r.Spec.Application.Name != "" && r.Spec.Application.Selector != nil {
		return nil, fmt.Errorf("Both Application name and Application selector cannot be used together")
	}
	return nil, nil
}

func (v *serviceClaimValidator) validateUpdate(old *ServiceClaim, new *ServiceClaim) error {

	if !reflect.DeepEqual(old.Spec, new.Spec) {
		return fmt.Errorf("Service Claim's Service Class Identity or Service Endpoint Definition Keys are not meant to be updated, Please delete the existing service claim")
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *serviceClaimValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	newServiceClaim, ok := newObj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", newObj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	oldServiceClaim, ok := oldObj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Old Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", oldObj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclaimlog.Info("validate update", "name", newServiceClaim.Name)
	return nil, v.validateUpdate(oldServiceClaim, newServiceClaim)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *serviceClaimValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*ServiceClaim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclaimlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
