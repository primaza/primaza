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

package authz

import (
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourcePermissions struct {
	Verbs    []string
	Group    string
	Version  string
	Resource string
	Name     string
}

func (r ResourcePermissions) toNamespacedPermissions(namespace string) []NamespacedPermission {
	pp := make([]NamespacedPermission, len(r.Verbs))
	for i, v := range r.Verbs {
		pp[i] = NamespacedPermission{
			Verb:      v,
			Group:     r.Group,
			Version:   r.Version,
			Resource:  r.Resource,
			Namespace: namespace,
			Name:      r.Name,
		}
	}
	return pp
}

type NamespacedPermission struct {
	Verb      string
	Group     string
	Version   string
	Resource  string
	Namespace string
	Name      string
}

func (p NamespacedPermission) String() string {
	if p.Name == "" {
		return fmt.Sprintf("%s %s.%s/%s in %s",
			p.Verb, p.Resource, p.Group, p.Version, p.Namespace)
	}

	return fmt.Sprintf("%s %s.%s/%s %s in %s",
		p.Verb, p.Resource, p.Group, p.Version, p.Name, p.Namespace)
}

func (np *NamespacedPermission) selfSubjectAccessReview() authorizationv1.SelfSubjectAccessReview {
	return authorizationv1.SelfSubjectAccessReview{
		ObjectMeta: metav1.ObjectMeta{Namespace: np.Namespace},
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:      np.Verb,
				Version:   np.Version,
				Group:     np.Group,
				Resource:  np.Resource,
				Namespace: np.Namespace,
				Name:      np.Name,
			},
		},
	}
}

type NamespacedPermissionsReport struct {
	Satisfied []NamespacedPermission
	Failed    []NamespacedPermission
	InError   map[NamespacedPermission]error
}

func (r *NamespacedPermissionsReport) AllSatisfied() bool {
	return len(r.Failed) == 0 && len(r.InError) == 0
}

func (r *NamespacedPermissionsReport) satisfied(np NamespacedPermission) {
	r.Satisfied = append(r.Satisfied, np)
}

func (r *NamespacedPermissionsReport) failed(np NamespacedPermission) {
	r.Failed = append(r.Failed, np)
}

func (r *NamespacedPermissionsReport) inError(np NamespacedPermission, err error) {
	if r.InError == nil {
		r.InError = map[NamespacedPermission]error{}
	}

	r.InError[np] = err
}
