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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/phylee/application-operator/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.lee.cn,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.lee.cn,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.lee.cn,resources=applications/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the current cluster state toward the desired state described
// by an Application: it creates/updates/deletes raw Pods to match
// ApplicationSpec.Replicas, deriving each Pod's spec from
// ApplicationSpec.Template.
//
// Caveats vs. using a Deployment: there is no rolling update, no self-healing
// on container crash, and no stable network identity. The controller will
// recreate missing Pods on the next reconcile (e.g. when the Application
// itself changes), but not automatically on transient pod failures.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("application", req.NamespacedName)

	// 1. Fetch the Application; ignore not-found (object was deleted).
	app := &appsv1.Application{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Resolve desired replica count.
	replicas := int32(1)
	if app.Spec.Replicas != nil {
		replicas = *app.Spec.Replicas
	}

	// 3. List existing Pods owned by this Application (matched by label).
	var podList corev1.PodList
	if err := r.List(ctx, &podList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{appLabelKey: app.Name},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("list pods: %w", err)
	}
	existing := make(map[string]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		existing[podList.Items[i].Name] = &podList.Items[i]
	}

	// 4. Create or update pods for indexes [0, replicas).
	//
	// Pods are special: K8s disallows most spec updates, so we only ever
	// create new pods. Spec drift (any change to the container definition)
	// is detected via a spec-hash annotation — a stale pod is deleted so
	// the next iteration recreates it with the desired spec.
	desiredHash := containerSpecHash(app.Spec.Template.Containers)
	for i := int32(0); i < replicas; i++ {
		podName := fmt.Sprintf("%s-%d", app.Name, i)

		if cur, ok := existing[podName]; ok {
			// Pod exists. If its spec hash matches, it is up to date.
			if cur.Annotations != nil && cur.Annotations[specHashAnnotation] == desiredHash {
				delete(existing, podName) // desired, don't delete
				continue
			}
			// Spec drift: delete and let the next reconcile recreate.
			log.Info("Pod spec drift, deleting for recreate", "pod", podName)
			if err := r.Delete(ctx, cur); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("delete stale pod %s: %w", podName, err)
			}
			delete(existing, podName)
			continue
		}

		// Create the pod from scratch.
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        podName,
				Namespace:   app.Namespace,
				Labels:      map[string]string{appLabelKey: app.Name},
				Annotations: map[string]string{specHashAnnotation: desiredHash},
			},
			Spec: corev1.PodSpec{
				Containers: appContainersToCore(app.Spec.Template.Containers),
			},
		}
		if err := controllerutil.SetControllerReference(app, pod, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("set owner ref on pod %s: %w", podName, err)
		}
		if err := r.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("create pod %s: %w", podName, err)
		}
	}

	// 5. Delete any extra pods left over from a previous (larger) replica count.
	for name, p := range existing {
		if err := r.Delete(ctx, p); err != nil {
			return ctrl.Result{}, fmt.Errorf("delete excess pod %s: %w", name, err)
		}
		log.Info("Deleted excess pod", "pod", name)
	}

	// 6. Re-list pods to compute status.
	if err := r.List(ctx, &podList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{appLabelKey: app.Name},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("list pods for status: %w", err)
	}

	var ready int32
	var running int32
	for i := range podList.Items {
		pod := &podList.Items[i]
		// Only count non-terminating pods.
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		running++
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ready++
				break
			}
		}
	}

	app.Status.Replicas = running
	app.Status.ReadyReplicas = ready

	// Set Available condition.
	availableStatus := metav1.ConditionFalse
	availableReason := "PodsNotReady"
	availableMsg := fmt.Sprintf("%d/%d pods are ready", ready, replicas)
	if ready >= replicas {
		availableStatus = metav1.ConditionTrue
		availableReason = "AllPodsReady"
		if replicas > 0 {
			availableMsg = fmt.Sprintf("All %d pods are ready", replicas)
		} else {
			availableMsg = "Scaled to zero"
		}
	}
	meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
		Type:               conditionAvailable,
		Status:             availableStatus,
		ObservedGeneration: app.Generation,
		Reason:             availableReason,
		Message:            availableMsg,
	})

	if err := r.Status().Update(ctx, app); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}

	log.Info("Pods reconciled", "replicas", replicas, "ready", ready)
	return ctrl.Result{}, nil
}

// appLabelKey is the label used to associate pods with their parent Application.
const appLabelKey = "apps.lee.cn/app"

// specHashAnnotation is set on pods to detect spec drift. When the desired
// container spec changes, the hash no longer matches and the controller
// deletes the stale pod so a new one is created.
const specHashAnnotation = "apps.lee.cn/spec-hash"

// condition types
const (
	conditionAvailable = "Available"
)

// appContainersToCore converts the API's slim container definition into a
// corev1.Container suitable for a Pod spec.
func appContainersToCore(in []appsv1.ApplicationContainer) []corev1.Container {
	out := make([]corev1.Container, 0, len(in))
	for _, c := range in {
		out = append(out, corev1.Container{
			Name:            c.Name,
			Image:           c.Image,
			Command:         c.Command,
			Args:            c.Args,
			ImagePullPolicy: c.ImagePullPolicy,
			Ports:           c.Ports,
			Env:             c.Env,
			EnvFrom:         c.EnvFrom,
			Resources:       c.Resources,
			LivenessProbe:   c.LivenessProbe,
			ReadinessProbe:  c.ReadinessProbe,
		})
	}
	return out
}

// containerSpecHash returns a hex-encoded SHA-256 hash of the JSON-marshalled
// container specs. The hash ignores runtime-only fields so two logically
// equivalent specs produce the same hash.
func containerSpecHash(in []appsv1.ApplicationContainer) string {
	if len(in) == 0 {
		return ""
	}
	data, err := json.Marshal(in)
	if err != nil {
		// Should never happen — our types are always JSON-marshalable.
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Application{}).
		Named("application").
		Complete(r)
}
