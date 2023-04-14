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
	"fmt"
	"strings"
)

var (
	agentRoles = map[NamespaceType][]string{
		ServiceNamespaceType:     {"primaza-reporter"},
		ApplicationNamespaceType: {"primaza-claimer"},
	}
)

func getAgentRoleNames(agentKind NamespaceType) []string {
	if rr, ok := agentRoles[agentKind]; ok {
		return rr
	}
	return nil
}

func bakeRoleBindingName(role, ceName, namespace string) string {
	tr, _ := strings.CutPrefix(role, "primaza:")
	return fmt.Sprintf("%s-%s-%s", tr, ceName, namespace)
}
