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

package controlplane

import (
	"context"

	"github.com/primaza/primaza/pkg/authz"
	wauthz "github.com/primaza/primaza/pkg/primaza/workercluster/authz"
	"k8s.io/client-go/rest"
)

type AgentPermissionsChecker interface {
	TestPermissions(ctx context.Context, namespaces []string) (AgentPermissionsCheckReport, error)
	CheckExcessPermission(ctx context.Context, namespaces []string) ([]string, error)
}

type AgentPermissionsCheckReport map[string]authz.NamespacedPermissionsReport

func NewAgentAppPermissionsChecker(cfg *rest.Config) AgentPermissionsChecker {
	return &agentPermissionsChecker{
		cfg:                    cfg,
		getResourcePermissions: wauthz.GetAgentAppRequiredPermissions,
		getPermissionList:      wauthz.GetAppPermissionList,
	}
}

func NewAgentSvcPermissionsChecker(cfg *rest.Config) AgentPermissionsChecker {
	return &agentPermissionsChecker{
		cfg:                    cfg,
		getResourcePermissions: wauthz.GetAgentSvcRequiredPermissions,
		getPermissionList:      wauthz.GetSvcPermissionList,
	}
}

type agentPermissionsChecker struct {
	cfg                    *rest.Config
	getResourcePermissions func() []authz.ResourcePermissions
	getPermissionList      func() []authz.Permission
}

func (c *agentPermissionsChecker) TestPermissions(ctx context.Context, namespaces []string) (AgentPermissionsCheckReport, error) {
	pp := c.getResourcePermissions()
	return authz.TestResourcePermissions(ctx, c.cfg, namespaces, pp)
}

func (c *agentPermissionsChecker) CheckExcessPermission(ctx context.Context, namespaces []string) ([]string, error) {
	pl := c.getPermissionList()
	return authz.AccessList(ctx, c.cfg, namespaces, pl)
}
