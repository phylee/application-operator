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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "github.com/phylee/application-operator/api/v1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("Application Webhook", func() {
	var (
		obj       *appsv1.Application
		oldObj    *appsv1.Application
		validator ApplicationCustomValidator
		defaulter ApplicationCustomDefaulter
	)

	BeforeEach(func() {
		obj = &appsv1.Application{}
		oldObj = &appsv1.Application{}
		validator = ApplicationCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = ApplicationCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating Application under Defaulting Webhook", func() {
		It("Should default deployment replicas to 3 when replicas is omitted", func() {
			By("calling the Default method to apply defaults")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("checking that deployment replicas is set")
			Expect(obj.Spec.Deployment.Replicas).NotTo(BeNil())
			Expect(*obj.Spec.Deployment.Replicas).To(Equal(int32(3)))
		})

		It("Should preserve deployment replicas when replicas is provided", func() {
			replicas := int32(5)
			obj.Spec.Deployment.Replicas = &replicas

			By("calling the Default method to apply defaults")
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())

			By("checking that deployment replicas is unchanged")
			Expect(obj.Spec.Deployment.Replicas).NotTo(BeNil())
			Expect(*obj.Spec.Deployment.Replicas).To(Equal(replicas))
		})
	})

	Context("When creating or updating Application under Validating Webhook", func() {
		It("Should allow validation when replicas is omitted", func() {
			By("calling ValidateCreate with replicas omitted")
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("Should deny validation when replicas is greater than 10", func() {
			replicas := int32(11)
			obj.Spec.Deployment.Replicas = &replicas

			By("calling ValidateCreate with too many replicas")
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(MatchError("replicas cannot be greater than 10"))
		})
	})

})
