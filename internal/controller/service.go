package controller

import (
	"context"
	"reflect"

	apiv1 "github.com/phylee/application-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ApplicationReconciler) reconcileService(ctx context.Context, app *apiv1.Application) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	labels := applicationLabels(app)

	var svc = &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: app.Namespace,
		Name:      app.Name,
	}, svc)

	if err == nil {
		log.Info("Service already exists")

		updated := false
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}, svc); err != nil {
				return err
			}

			desiredSpec := desiredServiceSpec(app, labels)
			preserveServiceAllocatedFields(desiredSpec, svc.Spec)
			if reflect.DeepEqual(svc.Labels, labels) && reflect.DeepEqual(svc.Spec, *desiredSpec) {
				return nil
			}

			svc.Labels = labels
			svc.Spec = *desiredSpec
			if err := r.Update(ctx, svc); err != nil {
				return err
			}
			updated = true
			return nil
		}); err != nil {
			log.Error(err, "Failed to update Service")
			return ctrl.Result{}, err
		}
		if updated {
			log.Info("Updated Service", "name", svc.Name)
		}

		if reflect.DeepEqual(svc.Status, app.Status.Network) {
			return ctrl.Result{}, nil
		}

		app.Status.Network = svc.Status
		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "Failed to update Application status")
			return ctrl.Result{}, err
		}
		log.Info("Updated Application status")
		return ctrl.Result{}, nil
	}

	if !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	newSvc := &corev1.Service{}
	newSvc.SetName(app.Name)
	newSvc.SetNamespace(app.Namespace)
	newSvc.SetLabels(labels)
	newSvc.Spec = *desiredServiceSpec(app, labels)

	if err := ctrl.SetControllerReference(app, newSvc, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, newSvc); err != nil {
		log.Error(err, "Failed to create Service")
		return ctrl.Result{}, err
	}

	log.Info("Created Service", "name", newSvc.Name)
	return ctrl.Result{}, nil
}

func desiredServiceSpec(app *apiv1.Application, labels map[string]string) *corev1.ServiceSpec {
	spec := app.Spec.Service.DeepCopy()
	if spec.Selector == nil && spec.Type != corev1.ServiceTypeExternalName {
		spec.Selector = labels
	}
	return spec
}

func preserveServiceAllocatedFields(desired *corev1.ServiceSpec, existing corev1.ServiceSpec) {
	if desired.ClusterIP == "" {
		desired.ClusterIP = existing.ClusterIP
	}
	if len(desired.ClusterIPs) == 0 {
		desired.ClusterIPs = existing.ClusterIPs
	}
	if len(desired.IPFamilies) == 0 {
		desired.IPFamilies = existing.IPFamilies
	}
	if desired.IPFamilyPolicy == nil {
		desired.IPFamilyPolicy = existing.IPFamilyPolicy
	}

	for i := range desired.Ports {
		if desired.Ports[i].NodePort != 0 {
			continue
		}
		for _, existingPort := range existing.Ports {
			if servicePortsMatch(desired.Ports[i], existingPort) {
				desired.Ports[i].NodePort = existingPort.NodePort
				break
			}
		}
	}
}

func servicePortsMatch(desired corev1.ServicePort, existing corev1.ServicePort) bool {
	if desired.Name != "" || existing.Name != "" {
		return desired.Name == existing.Name
	}
	return desired.Port == existing.Port && desired.Protocol == existing.Protocol
}
