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
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/jsonpath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceclasslog = logf.Log.WithName("serviceclass-resource")

type serviceClassValidator struct {
	client client.Client
}

var _ admission.CustomValidator = &serviceClassValidator{}

func (r *ServiceClass) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&serviceClassValidator{
			client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-primaza-io-v1alpha1-serviceclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=primaza.io,resources=serviceclasses,verbs=create;update,versions=v1alpha1,name=vserviceclass.kb.io,admissionReviewVersions=v1

func (r *ServiceClassResource) ValidateMapping() field.ErrorList {
	errs := field.ErrorList{}
	names := map[string]struct{}{}
	childPath := field.NewPath("spec", "resource")
	for i, mapping := range r.ServiceEndpointDefinitionMappings.ResourceFields {
		path := childPath.Child("serviceEndpointDefinitionMapping").Index(i)
		j := jsonpath.New("")
		formatted := fmt.Sprintf("{%v}", mapping.JsonPath)
		if err := j.Parse(formatted); err != nil {
			errs = append(errs, field.Invalid(path.Child("jsonPath"), mapping.JsonPath, "Invalid JSONPath"))
		}
		if _, found := names[mapping.Name]; found {
			errs = append(errs, field.Duplicate(path.Child("name"), mapping.Name))
		} else {
			names[mapping.Name] = struct{}{}
		}
	}

	return errs
}

// ValidateCreate implements admission.CustomValidator
func (v *serviceClassValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*ServiceClass)
	if !ok {
		err := fmt.Errorf("Object is not a Service Class")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClass resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclasslog.Info("validate create", "name", r.Name)
	errs, err := v.IsDuplicateClass(ctx, *r)
	if err != nil {
		return nil, err
	}
	errs = append(errs, r.Spec.Resource.ValidateMapping()...)
	return nil, errs.ToAggregate()
}

// ValidateDelete implements admission.CustomValidator
func (v *serviceClassValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*ServiceClass)
	if !ok {
		err := fmt.Errorf("Object is not a Service Class")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClass resource", "gvk", obj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclasslog.Info("validate delete", "name", r.Name)
	return nil, nil // no validation
}

// ValidateUpdate implements admission.CustomValidator
func (v *serviceClassValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	newClass, ok := newObj.(*ServiceClass)
	if !ok {
		err := fmt.Errorf("Object is not a Service Class")
		serviceclasslog.Error(err, "Attempted to validate non-ServiceClass resource", "gvk", newObj.GetObjectKind().GroupVersionKind())
		return nil, err
	}

	serviceclasslog.Info("validate update", "name", newClass.Name)

	oldServiceClass, ok := oldObj.(*ServiceClass)
	if !ok {
		return nil, fmt.Errorf("Old object is not a ServiceClass")
	}

	errs := field.ErrorList{}
	childPath := field.NewPath("spec", "resource")
	if oldServiceClass.Spec.Resource.APIVersion != newClass.Spec.Resource.APIVersion {
		errs = append(errs, field.Invalid(childPath.Child("apiVersion"), newClass.Spec.Resource.APIVersion, "APIVersion is immutable"))
	}
	if oldServiceClass.Spec.Resource.Kind != newClass.Spec.Resource.Kind {
		errs = append(errs, field.Invalid(childPath.Child("kind"), newClass.Spec.Resource.Kind, "Kind is immutable"))
	}
	// a cheap way of doing data sets; struct{} is zero-sized, so we don't needlessly make allocations
	oldMappings := map[ServiceClassResourceFieldMapping]struct{}{}
	newMappings := map[ServiceClassResourceFieldMapping]struct{}{}
	for _, item := range oldServiceClass.Spec.Resource.ServiceEndpointDefinitionMappings.ResourceFields {
		oldMappings[item] = struct{}{}
	}
	for _, item := range newClass.Spec.Resource.ServiceEndpointDefinitionMappings.ResourceFields {
		newMappings[item] = struct{}{}
	}
	if !reflect.DeepEqual(oldMappings, newMappings) {
		errs = append(errs,
			field.Invalid(childPath.Child("serviceEndpointDefinitionMapping"),
				newClass.Spec.Resource.ServiceEndpointDefinitionMappings,
				"ServiceEndpointDefinitionMapping is immutable"))
	}
	errs = append(errs, newClass.Spec.Resource.ValidateMapping()...)
	list, err := v.IsDuplicateClass(ctx, *newClass)
	if err != nil {
		return nil, err
	}
	errs = append(errs, list...)

	return nil, errs.ToAggregate()
}

func (validator *serviceClassValidator) IsDuplicateClass(ctx context.Context, serviceClass ServiceClass) (field.ErrorList, error) {
	classList := ServiceClassList{}
	err := validator.client.List(ctx, &classList)
	if err != nil {
		// The list call failed; report as an error.
		return nil, err
	}

	serviceclasslog.Info("checking items", "items", classList)
	for _, item := range classList.Items {
		if serviceClass.Name != item.Name &&
			serviceClass.Spec.Resource.Kind == item.Spec.Resource.Kind &&
			serviceClass.Spec.Resource.APIVersion == item.Spec.Resource.APIVersion {
			// We found another ServiceClass that manages the same kind/apiVersion in this namespace, so report it as a match.
			return field.ErrorList{
				field.Forbidden(field.NewPath("spec", "resource"),
					fmt.Sprintf("Service Class %v already manages services of type %v.%v",
						item.Name,
						item.Spec.Resource.Kind,
						item.Spec.Resource.APIVersion))}, nil
		}
	}

	// no matches
	return nil, nil

}
