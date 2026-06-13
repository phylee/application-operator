/*
Copyright 2026.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "github.com/phylee/application-operator/api/v1"
)

var _ = Describe("Application Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		application := &appsv1.Application{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Application")
			err := k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				replicas := int32(1)
				resource := &appsv1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: appsv1.ApplicationSpec{
						Replicas: &replicas,
						Template: appsv1.ApplicationTemplate{
							Containers: []appsv1.ApplicationContainer{
								{
									Name:  "nginx",
									Image: "nginx:1.27-alpine",
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &appsv1.Application{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Application")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Application status was updated")
			Eventually(func(g Gomega) {
				updated := &appsv1.Application{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
				g.Expect(updated.Status.Replicas).To(BeNumerically(">", 0))
				g.Expect(updated.Status.Conditions).NotTo(BeEmpty())
				cond := updated.Status.Conditions[0]
				g.Expect(cond.Type).To(Equal("Available"))
				// Pods never become Ready in envtest (no kubelet), so
				// the condition is expected to be False with PodsNotReady.
				g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(cond.Reason).To(Equal("PodsNotReady"))
			}, "10s", "500ms").Should(Succeed())
		})
	})
})
