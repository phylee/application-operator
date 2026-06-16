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
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1 "github.com/phylee/application-operator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var applicationlog = logf.Log.WithName("application-resource")

const defaultDeploymentReplicas int32 = 3

// SetupApplicationWebhookWithManager registers the webhook for Application in the manager.
func SetupApplicationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &appsv1.Application{}).
		WithValidator(&ApplicationCustomValidator{}).
		WithDefaulter(&ApplicationCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-apps-lee-cn-v1-application,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.lee.cn,resources=applications,verbs=create;update,versions=v1,name=mapplication-v1.kb.io,admissionReviewVersions=v1

// ApplicationCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Application when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ApplicationCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Application.
func (d *ApplicationCustomDefaulter) Default(_ context.Context, obj *appsv1.Application) error {
	applicationlog.Info("Defaulting for Application", "name", obj.GetName())

	if obj.Spec.Deployment.Replicas == nil {
		obj.Spec.Deployment.Replicas = ptr.To(defaultDeploymentReplicas)
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-apps-lee-cn-v1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.lee.cn,resources=applications,verbs=create;update,versions=v1,name=vapplication-v1.kb.io,admissionReviewVersions=v1

// ApplicationCustomValidator struct is responsible for validating the Application resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ApplicationCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateCreate(_ context.Context, obj *appsv1.Application) (admission.Warnings, error) {
	applicationlog.Info("Validation for Application upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.
	return v.validateApplication(obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *appsv1.Application) (admission.Warnings, error) {
	applicationlog.Info("Validation for Application upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return v.validateApplication(newObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateDelete(_ context.Context, obj *appsv1.Application) (admission.Warnings, error) {
	applicationlog.Info("Validation for Application upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

func (v *ApplicationCustomValidator) validateApplication(obj *appsv1.Application) (admission.Warnings, error) {
	if obj.Spec.Deployment.Replicas == nil {
		return nil, nil
	}

	if *obj.Spec.Deployment.Replicas > 10 {
		return nil, fmt.Errorf("replicas cannot be greater than 10")
	}
	return nil, nil
}
