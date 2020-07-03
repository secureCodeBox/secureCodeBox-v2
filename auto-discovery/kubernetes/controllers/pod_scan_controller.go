/*


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
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	targetsv1 "github.com/secureCodeBox/secureCodeBox-v2-alpha/operator/apis/targets/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodScanReconciler reconciles a DeleteMe object
type PodScanReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking,resources=pod,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking,resources=pod/status,verbs=get
// +kubebuilder:rbac:groups=networking,resources=secret,verbs=get

// Reconcile compares the Ingress object against the state of the cluster and updates both if needed
func (r *PodScanReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log

	log.Info("Something happened to a pod", "pod", req.Name, "namespace", req.Namespace)

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		log.V(7).Info("Unable to fetch Pod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		r.Log.Info("No Container Status yet")
		return ctrl.Result{}, nil
	}

	containerStatus := pod.Status.ContainerStatuses[0]

	if containerStatus.ImageID == "" {
		r.Log.Info("Image Id is empty")
		return ctrl.Result{}, nil
	}

	var fullImageName string
	if strings.HasPrefix(containerStatus.ImageID, "docker-pullable://") {
		// Extract the fullImageName from the following format "docker-pullable://scbexperimental/parser-nmap@sha256:f953..."
		fullImageName = containerStatus.ImageID[18:]
	} else {
		return ctrl.Result{}, fmt.Errorf("Unexpected image reference format '%s'", containerStatus.ImageID)
	}

	imageSegments := strings.Split(fullImageName, "@")
	imageName := strings.NewReplacer(
		"/", "-",
		":", "-",
	).Replace(imageSegments[0])
	prefixedDigest := imageSegments[1]

	var truncatedDigest string
	if strings.HasPrefix(prefixedDigest, "sha256:") {
		// Only keep actual digest
		// Also truncate the last char as we want to store it in a k8s label which only allows 63 chars and a sha256 digest ist 64 chars long 🤦‍
		// Example from "sha256:f953bc6c5446c20ace8787a1956c2e46a2556cc7a37ef7fc0dda7b11dd87f73d"
		// What is kept: "f953bc6c5446c20ace8787a1956c2e46a2556cc7a37ef7fc0dda7b11dd87f73"
		truncatedDigest = prefixedDigest[7:70]
	} else {
		return ctrl.Result{}, errors.New("Unexpected image digest format")
	}

	imageScan := targetsv1.ContainerImage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", imageName, truncatedDigest[0:6]),
			Namespace: req.Namespace,
			Labels: map[string]string{
				"image-digest.experimental.securecodebox.io": truncatedDigest,
			},
		},
		Spec: targetsv1.ContainerImageSpec{
			Image: fullImageName,
		},
	}

	r.Create(ctx, &imageScan)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller and initializes every thing it needs
func (r *PodScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&targetsv1.ContainerImage{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		containerImage := rawObj.(*targetsv1.ContainerImage)
		owner := metav1.GetControllerOf(containerImage)
		if owner == nil {
			return nil
		}

		// ...make sure it's a Pod...
		if owner.APIVersion != "v1" || owner.Kind != "Pod" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	isInDemoNamespaceFilter := predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}
			return event.Meta.GetNamespace() == "juice-shop" || event.Meta.GetNamespace() == "bodgeit"
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}
			return event.Meta.GetNamespace() == "juice-shop" || event.Meta.GetNamespace() == "bodgeit"
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			if val, ok := event.MetaNew.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}
			return event.MetaNew.GetNamespace() == "juice-shop" || event.MetaNew.GetNamespace() == "bodgeit"
		},
		GenericFunc: func(event event.GenericEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}
			return event.Meta.GetNamespace() == "juice-shop" || event.Meta.GetNamespace() == "bodgeit"
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(isInDemoNamespaceFilter).
		Complete(r)
}
