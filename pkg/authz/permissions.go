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
	"strings"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Permission struct {
	APIGroups     []string
	Resources     []string
	ResourceNames []string
	Namespace     string
	Name          string
	Verbs         []string
}

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

func createUnwantedPermissions(resourceRules []authzv1.ResourceRule, ns string, definedRules map[[4]string]int) []string {
	unwantedPermissions := []string{}
	for _, rr := range resourceRules {
		APIGroups := rr.APIGroups
		if len(APIGroups) == 0 {
			APIGroups = []string{""}
		}
		ResourceNames := rr.ResourceNames
		if len(ResourceNames) == 0 {
			ResourceNames = []string{""}
		}
		for _, ag := range APIGroups {
			for _, rs := range rr.Resources {
				for _, rn := range ResourceNames {
					for _, vb := range rr.Verbs {
						if ag == "*" || rs == "*" {
							unwantedPermissions = append(
								unwantedPermissions,
								fmt.Sprintf(
									"Excess permission: namespace=%s,resource=%s,resource-name=%s,verbs=%s",
									ns, strings.Join([]string{rs, ag}, "."), rn, vb,
								),
							)
						}
						if _, ok := definedRules[[4]string{ag, rs, rn, vb}]; !ok {
							unwantedPermissions = append(
								unwantedPermissions,
								fmt.Sprintf(
									"Excess permission: namespace=%s,resource=%s,resource-name=%s,verbs=%s",
									ns, strings.Join([]string{rs, ag}, "."), rn, vb,
								),
							)
						}
					}
				}
			}
		}
	}
	return unwantedPermissions
}
func createUnwantedPermissionsList(ctx context.Context, cfg *rest.Config, ns string, definedRules map[[4]string]int) ([]string, error) {
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return []string{}, fmt.Errorf("error creating the client: %w", err)
	}

	ssrr := authzv1.SelfSubjectRulesReview{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns},
		Spec:       authzv1.SelfSubjectRulesReviewSpec{Namespace: ns},
	}

	r, err := c.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &ssrr, metav1.CreateOptions{})
	if err != nil {
		return []string{}, err
	}
	resourceRules := r.Status.ResourceRules

	unwantedPermissions := createUnwantedPermissions(resourceRules, ns, definedRules)

	return unwantedPermissions, nil
}

func AccessList(ctx context.Context, cfg *rest.Config, namespaces []string, pl []Permission) ([]string, error) {
	unwantedPermissions := []string{}
	for _, ns := range namespaces {

		definedRules := make(map[[4]string]int)
		for _, p := range pl {
			APIGroups := p.APIGroups
			if len(APIGroups) == 0 {
				APIGroups = []string{""}
			}
			ResourceNames := p.ResourceNames
			if len(ResourceNames) == 0 {
				ResourceNames = []string{""}
			}
			for _, ag := range APIGroups {
				for _, rs := range p.Resources {
					for _, rn := range ResourceNames {
						for _, vb := range p.Verbs {
							definedRules[[4]string{ag, rs, rn, vb}] = 1
						}
					}
				}
			}
		}

		up, err := createUnwantedPermissionsList(ctx, cfg, ns, definedRules)
		if err != nil {
			return []string{}, err
		}
		unwantedPermissions = append(unwantedPermissions, up...)
	}

	return unwantedPermissions, nil
}
