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
var claimlog = logf.Log.WithName("claim-resource")

type claimValidator struct {
	client client.Client
}

func (r *Claim) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&claimValidator{
			client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-primaza-io-v1alpha1-claim,mutating=false,failurePolicy=fail,sideEffects=None,groups=primaza.io,resources=claims,verbs=create;update,versions=v1alpha1,name=vclaim.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &claimValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *claimValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*Claim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	claimlog.Info("validate create", "name", r.Name)
	return v.validate(r)
}

func (v *claimValidator) validate(r *Claim) (admission.Warnings, error) {
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

func (v *claimValidator) validateUpdate(old *Claim, new *Claim) error {

	if new.Spec.ApplicationClusterContext != nil && new.Spec.EnvironmentTag != "" {
		return fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be used together")
	}
	if new.Spec.ApplicationClusterContext == nil && new.Spec.EnvironmentTag == "" {
		return fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be empty")
	}
	if new.Spec.Application.Name != "" && new.Spec.Application.Selector != nil {
		return fmt.Errorf("Both Application name and Application selector cannot be used together")
	}
	if !reflect.DeepEqual(old.Spec.ServiceClassIdentity, new.Spec.ServiceClassIdentity) || !reflect.DeepEqual(old.Spec.ServiceEndpointDefinitionKeys, new.Spec.ServiceEndpointDefinitionKeys) {
		return fmt.Errorf("Service Claim's Service Class Identity or Service Endpoint Definition Keys are not meant to be updated, Please delete the existing service claim")
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *claimValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	newServiceClaim, ok := newObj.(*Claim)
	if !ok {
		err := fmt.Errorf("Object is not a  Claim")
		serviceclasslog.Error(err, "Attempted to validate non-Claim resource", "gvk", newObj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	oldServiceClaim, ok := oldObj.(*Claim)
	if !ok {
		err := fmt.Errorf("Old Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", oldObj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	claimlog.Info("validate update", "name", newServiceClaim.Name)
	return nil, v.validateUpdate(oldServiceClaim, newServiceClaim)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *claimValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*Claim)
	if !ok {
		err := fmt.Errorf("Object is not a Service Claim")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClaim resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	claimlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
