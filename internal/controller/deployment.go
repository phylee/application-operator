package controller

import (
	"context"
	"maps"
	"reflect"

	apiv1 "github.com/phylee/application-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ApplicationReconciler) reconcileDeployment(ctx context.Context, app *apiv1.Application) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	labels := applicationLabels(app)

	var dp = &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: app.Namespace,
		Name:      app.Name,
	}, dp)

	if err == nil {
		log.Info("Deployment already exists")

		updated := false
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}, dp); err != nil {
				return err
			}

			desiredSpec := desiredDeploymentSpec(app, labels)
			desiredSpec.Selector = dp.Spec.Selector
			if reflect.DeepEqual(dp.Labels, labels) && reflect.DeepEqual(dp.Spec, *desiredSpec) {
				return nil
			}

			dp.Labels = labels
			dp.Spec = *desiredSpec
			if err := r.Update(ctx, dp); err != nil {
				return err
			}
			updated = true
			return nil
		}); err != nil {
			log.Error(err, "Failed to update Deployment")
			return ctrl.Result{}, err
		}
		if updated {
			log.Info("Updated Deployment", "name", dp.Name)
		}

		if reflect.DeepEqual(dp.Status, app.Status.Workflow) {
			return ctrl.Result{}, nil
		}

		app.Status.Workflow = dp.Status
		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "Failed to update Application status")
			return ctrl.Result{}, err
		}
		log.Info("Updated Application status")
		return ctrl.Result{}, nil
	}

	if !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	newDp := &appsv1.Deployment{}
	newDp.SetName(app.Name)
	newDp.SetNamespace(app.Namespace)
	newDp.SetLabels(labels)
	newDp.Spec = *desiredDeploymentSpec(app, labels)

	if err := ctrl.SetControllerReference(app, newDp, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, newDp); err != nil {
		log.Error(err, "Failed to create Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Created Deployment", "name", newDp.Name)
	return ctrl.Result{}, nil
}

func desiredDeploymentSpec(app *apiv1.Application, labels map[string]string) *appsv1.DeploymentSpec {
	spec := app.Spec.Deployment.DeepCopy()
	if spec.Selector == nil {
		spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	}
	if spec.Template.Labels == nil {
		spec.Template.Labels = map[string]string{}
	}
	maps.Copy(spec.Template.Labels, spec.Selector.MatchLabels)
	return spec
}
