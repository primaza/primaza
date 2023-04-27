package controlplane

import (
	"github.com/primaza/primaza/pkg/primaza/constants"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func getRoleBindingsLabelSelectorOrDie(ceName string, namespaceType NamespaceType) labels.Selector {
	newEqualRequirementOrDie := func(key, value string) *labels.Requirement {
		lr, err := labels.NewRequirement(key, selection.Equals, []string{value})
		if err != nil {
			// not expecting to happen
			panic(err)
		}
		return lr
	}

	return labels.NewSelector().
		Add(*newEqualRequirementOrDie("app", "primaza")).
		Add(*newEqualRequirementOrDie(constants.PrimazaClusterEnvironmentLabel, ceName)).
		Add(*newEqualRequirementOrDie(constants.PrimazaNamespaceTypeLabel, string(namespaceType)))
}

func bakeRoleBindingsLabels(ceName, tenant, namespace string, namespaceType NamespaceType) map[string]string {
	return map[string]string{
		"app":                                    "primaza",
		constants.PrimazaTenantLabel:             tenant,
		constants.PrimazaClusterEnvironmentLabel: ceName,
		constants.PrimazaNamespaceTypeLabel:      string(namespaceType),
		constants.PrimazaNamespaceLabel:          namespace,
	}
}
