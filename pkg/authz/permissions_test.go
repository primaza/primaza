package authz

import (
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
)

func TestCreateUnwantedPermissions(t *testing.T) {
	resourceRules := []authzv1.ResourceRule{
		{
			Verbs:         []string{"get", "create"},
			APIGroups:     []string{""},
			Resources:     []string{"secrets"},
			ResourceNames: []string{},
		}}
	ns := "applications"
	k := [4]string{"", "secrets", "", "get"}
	definedRules := map[[4]string]int{k: 1}
	unwantedPermissions := createUnwantedPermissions(resourceRules, ns, definedRules)
	if len(unwantedPermissions) != 1 {
		t.Errorf("Wrong length: %v", unwantedPermissions)
	}
	o := "Excess permission: namespace=applications,resource=secrets.,resource-name=,verbs=create"
	if unwantedPermissions[0] != o {
		t.Errorf("Wrong output: %v", unwantedPermissions)
	}
}
