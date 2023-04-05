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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newServiceClaim(name, namespace string, spec ServiceClaimSpec) ServiceClaim {
	return ServiceClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

var _ = Describe("Webhook tests", func() {
	Context("When creating ServiceClaim with ApplicationClusterContext and EnvironmentTag", func() {
		It("should an error saying the resource cannot be created", func() {
			var validator serviceClaimValidator
			schemeBuilder, err := SchemeBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			validator = serviceClaimValidator{
				client: fake.NewClientBuilder().
					WithScheme(schemeBuilder).
					WithLists(&ServiceClaimList{}).
					Build(),
			}
			scacc := ServiceClaimApplicationClusterContext{}
			serviceClaim := newServiceClaim("spam", "eggs",
				ServiceClaimSpec{
					EnvironmentTag:            "prod",
					ApplicationClusterContext: &scacc,
				},
			)

			expected := fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be used together")
			Expect(validator.ValidateCreate(context.Background(), &serviceClaim)).To(Equal(expected))
		})
	})

	Context("When creating ServiceClaim with empty ApplicationClusterContext and EnvironmentTag", func() {
		It("should an error saying the resource cannot be created", func() {
			var validator serviceClaimValidator
			schemeBuilder, err := SchemeBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			validator = serviceClaimValidator{
				client: fake.NewClientBuilder().
					WithScheme(schemeBuilder).
					WithLists(&ServiceClaimList{}).
					Build(),
			}
			serviceClaim := newServiceClaim("spam", "eggs",
				ServiceClaimSpec{
					EnvironmentTag:            "",
					ApplicationClusterContext: nil,
				},
			)

			expected := fmt.Errorf("Both ApplicationClusterContext and EnvironmentTag cannot be empty")
			Expect(validator.ValidateCreate(context.Background(), &serviceClaim)).To(Equal(expected))
		})
	})

	Context("When creating ServiceClaim with Application name and Application selector", func() {
		It("should an error saying the resource cannot be created", func() {
			var validator serviceClaimValidator
			schemeBuilder, err := SchemeBuilder.Build()
			Expect(err).NotTo(HaveOccurred())

			validator = serviceClaimValidator{
				client: fake.NewClientBuilder().
					WithScheme(schemeBuilder).
					WithLists(&ServiceClaimList{}).
					Build(),
			}
			as := ApplicationSelector{
				Name:     "some-name",
				Selector: &metav1.LabelSelector{},
			}
			serviceClaim := newServiceClaim("spam", "eggs",
				ServiceClaimSpec{
					Application:    as,
					EnvironmentTag: "prod",
				},
			)

			expected := fmt.Errorf("Both Application name and Application selector cannot be used together")
			Expect(validator.ValidateCreate(context.Background(), &serviceClaim)).To(Equal(expected))
		})
	})

})
