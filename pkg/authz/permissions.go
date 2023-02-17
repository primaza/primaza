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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestResourcePermissions(ctx context.Context, cfg *rest.Config, namespaces []string, permissions []ResourcePermissions) (map[string]NamespacedPermissionsReport, error) {
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating the client: %w", err)
	}

	return checkPermissions(ctx, c, namespaces, permissions), nil
}

func checkPermissions(ctx context.Context, c *kubernetes.Clientset, namespaces []string, permissions []ResourcePermissions) map[string]NamespacedPermissionsReport {
	rr := map[string]NamespacedPermissionsReport{}
	for _, ns := range namespaces {
		rr[ns] = checkPermissionsInNamespace(ctx, c, ns, permissions)
	}
	return rr
}

func checkPermissionsInNamespace(ctx context.Context, c *kubernetes.Clientset, namespace string, permissions []ResourcePermissions) NamespacedPermissionsReport {
	l := log.FromContext(ctx)
	r := NamespacedPermissionsReport{}
	for _, p := range permissions {
		for _, np := range p.toNamespacedPermissions(namespace) {
			ok, err := checkAccess(ctx, c, np)

			switch {
			case err != nil:
				l.Error(err, "checking permission", "permission", np)
				r.inError(np, err)
				continue
			case !ok:
				l.Info("permission not granted", "permission", np)
				r.failed(np)
				continue
			case ok:
				l.Info("permission granted", "permission", np)
				r.satisfied(np)
				continue
			}
		}
	}
	return r
}

func checkAccess(ctx context.Context, c *kubernetes.Clientset, np NamespacedPermission) (bool, error) {
	sar := np.selfSubjectAccessReview()

	r, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return r.Status.Allowed, nil
}
