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

package controllers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/primaza/primaza/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func genTestName(oldState v1alpha1.RegisteredServiceState, newState v1alpha1.RegisteredServiceState, jobCompletion batchv1.JobConditionType) string {
	return fmt.Sprintf("%s + %s => %s", oldState, jobCompletion, newState)
}

var _ = Describe("Registered Service reconciler tests", func() {
	Describe("Healthcheck tests", func() {
		var (
			client         client.Client
			namespace      string
			rsController   RegisteredServiceReconciler
			rs             v1alpha1.RegisteredService
			ctx            context.Context
			namespacedName types.NamespacedName
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "bar"
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}

			rs = v1alpha1.RegisteredService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
					UID:       "9749f39d-6049-4fa3-bfc9-c46aca534f3f",
				},
				Spec: v1alpha1.RegisteredServiceSpec{
					Constraints: &v1alpha1.RegisteredServiceConstraints{},
					HealthCheck: &v1alpha1.HealthCheck{
						Container: v1alpha1.HealthCheckContainer{
							Image:   "alpine:latest",
							Command: []string{"sleep", "10"},
							Minutes: 1,
						},
					},
					SLA:                       "",
					ServiceClassIdentity:      []v1alpha1.ServiceClassIdentityItem{},
					ServiceEndpointDefinition: []v1alpha1.ServiceEndpointDefinitionItem{},
				},
				Status: v1alpha1.RegisteredServiceStatus{},
			}
			namespacedName = types.NamespacedName{
				Namespace: rs.Namespace,
				Name:      rs.Name,
			}

			scheme := runtime.NewScheme()
			err := v1alpha1.AddToScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = batchv1.AddToScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
			err = corev1.AddToScheme(scheme)
			Expect(err).NotTo(HaveOccurred())

			client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&ns, &rs).
				WithStatusSubresource(&rs).
				Build()

			err = client.Get(ctx, namespacedName, &rs)
			Expect(err).NotTo(HaveOccurred())

			rsController = RegisteredServiceReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}
		})

		It("should set state to available when a healthcheck is undefined", func() {
			_, err := ctrl.CreateOrUpdate(ctx, rsController.Client, &rs, func() error {
				rs.Spec.HealthCheck = nil
				rs.Status.State = ""
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = rsController.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			err = client.Get(ctx, namespacedName, &rs)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.Status.State).To(Equal(v1alpha1.RegisteredServiceStateAvailable))
		})

		It("should set state to unknown when a healthcheck is defined", func() {
			_, err := rsController.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			err = client.Get(ctx, namespacedName, &rs)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.Status.State).To(Equal(v1alpha1.RegisteredServiceStateUnknown))
		})

		It("should remove old healthcheck cronjobs when a healthcheck is removed", func() {
			cronjob1 := batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cronjob-1",
					Namespace: namespace,
				},
			}
			cronjob2 := cronjob1.DeepCopy()
			cronjob2.Name = "cronjob-2"
			err := controllerutil.SetOwnerReference(&rs, cronjob2, client.Scheme())
			Expect(err).NotTo(HaveOccurred())

			err = client.Create(ctx, &cronjob1)
			Expect(err).NotTo(HaveOccurred())
			err = client.Create(ctx, cronjob2)
			Expect(err).NotTo(HaveOccurred())

			rs.Spec.HealthCheck = nil
			err = client.Update(ctx, &rs)
			Expect(err).NotTo(HaveOccurred())

			result, err := rsController.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			cronjobs := batchv1.CronJobList{}
			err = client.List(ctx, &cronjobs)
			Expect(err).NotTo(HaveOccurred())
			Expect(cronjobs.Items).To(ConsistOf(cronjob1))
		})

		DescribeTable("Healthcheck state transitions",
			func(oldState v1alpha1.RegisteredServiceState, newState v1alpha1.RegisteredServiceState, jobCompletion batchv1.JobConditionType) {
				_, err := ctrl.CreateOrUpdate(ctx, client, &rs, func() error {
					rs.Status.State = oldState
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
				err = rsController.registerHealthcheck(ctx, &rs)
				Expect(err).NotTo(HaveOccurred())

				cronjob := batchv1.CronJob{}
				err = client.Get(ctx, types.NamespacedName{Name: rs.Name, Namespace: rs.Namespace}, &cronjob)
				Expect(err).NotTo(HaveOccurred())

				// fake a job, since it never gets created
				var succeeded int32 = 0
				var failed int32 = 0
				if jobCompletion == batchv1.JobComplete {
					succeeded = 1
				} else if jobCompletion == batchv1.JobFailed {
					failed = 1
				}
				job := batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cronjob.Name,
						Namespace: namespace,
					},
					Spec: cronjob.Spec.JobTemplate.Spec,
					Status: batchv1.JobStatus{
						Conditions: []batchv1.JobCondition{
							{
								Type:               jobCompletion,
								Status:             corev1.ConditionTrue,
								LastProbeTime:      metav1.Now(),
								LastTransitionTime: metav1.Now(),
								Reason:             "Success",
								Message:            "Success",
							},
						},
						Succeeded: succeeded,
						Failed:    failed,
					},
				}
				err = controllerutil.SetControllerReference(&cronjob, &job, client.Scheme())
				Expect(err).NotTo(HaveOccurred())
				err = client.Create(ctx, &job)
				Expect(err).NotTo(HaveOccurred())

				err = rsController.handleHealthcheck(ctx, &rs)

				// assert successful reconciliation
				Expect(err).NotTo(HaveOccurred())

				// assert state transition
				Expect(rs.Status.State).To(Equal(newState))
			},

			Entry(genTestName, v1alpha1.RegisteredServiceStateUnknown, v1alpha1.RegisteredServiceStateAvailable, batchv1.JobComplete),
			Entry(genTestName, v1alpha1.RegisteredServiceStateUnknown, v1alpha1.RegisteredServiceStateUnknown, batchv1.JobSuspended),
			Entry(genTestName, v1alpha1.RegisteredServiceStateUnknown, v1alpha1.RegisteredServiceStateUnreachable, batchv1.JobFailed),
			Entry(genTestName, v1alpha1.RegisteredServiceStateAvailable, v1alpha1.RegisteredServiceStateAvailable, batchv1.JobComplete),
			Entry(genTestName, v1alpha1.RegisteredServiceStateAvailable, v1alpha1.RegisteredServiceStateUnknown, batchv1.JobSuspended),
			Entry(genTestName, v1alpha1.RegisteredServiceStateAvailable, v1alpha1.RegisteredServiceStateUnreachable, batchv1.JobFailed),
			Entry(genTestName, v1alpha1.RegisteredServiceStateClaimed, v1alpha1.RegisteredServiceStateClaimed, batchv1.JobComplete),
			Entry(genTestName, v1alpha1.RegisteredServiceStateClaimed, v1alpha1.RegisteredServiceStateUnknown, batchv1.JobSuspended),
			Entry(genTestName, v1alpha1.RegisteredServiceStateClaimed, v1alpha1.RegisteredServiceStateUnreachable, batchv1.JobFailed),
			Entry(genTestName, v1alpha1.RegisteredServiceStateUnreachable, v1alpha1.RegisteredServiceStateAvailable, batchv1.JobComplete),
			Entry(genTestName, v1alpha1.RegisteredServiceStateUnreachable, v1alpha1.RegisteredServiceStateUnknown, batchv1.JobSuspended),
			Entry(genTestName, v1alpha1.RegisteredServiceStateUnreachable, v1alpha1.RegisteredServiceStateUnreachable, batchv1.JobFailed),
		)
	})
})
