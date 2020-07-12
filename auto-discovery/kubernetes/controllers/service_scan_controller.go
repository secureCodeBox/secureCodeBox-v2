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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	targetsv1 "github.com/secureCodeBox/secureCodeBox-v2-alpha/operator/apis/targets/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceScanReconciler reconciles a Service object
type ServiceScanReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=service,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=service/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pod,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pod/status,verbs=get

// Reconcile compares the Service object against the state of the cluster and updates both if needed
func (r *ServiceScanReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log

	log.Info("Something happened to a service", "service", req.Name, "namespace", req.Namespace)

	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		log.V(7).Info("Unable to fetch Service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Got Service", "service", service.Name, "resourceVersion", service.ResourceVersion)

	// Checking if the service got likely something to do with http...
	if len(getLikelyHTTPPorts(service)) == 0 {
		log.Info("Services doesn't seem to have a http / https port")
		// No port which has likely to do anything with http. No need to schedule a requeue until the service gets updated
		return ctrl.Result{}, nil
	}

	var pods corev1.PodList
	r.List(ctx, &pods, client.MatchingLabels(service.Spec.Selector))

	log.Info("Got Pods for Service", "pods", len(pods.Items))

	podDigests := map[string]bool{}

	for _, pod := range pods.Items {
		digest := getShaHashForPod(pod)
		if digest == nil {
			continue
		}

		podDigests[*digest] = true
	}

	if len(podDigests) != 1 {
		// Pods for Service don't all have the same digest.
		// Probably currently updating. Checking again in a few seconds.
		log.Info("Services Pods Digests don't all match. Deployment is probably currently under way. Waiting for it to finish.")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	// Get the podDigest from the map.
	// The map should only contain one entry at this point.
	var podDigest string
	for key := range podDigests {
		podDigest = key
	}

	// Pod for the Service all have the same digest
	// Checking if we already have run a scan against this version

	var hosts targetsv1.HostList

	r.Client.List(ctx, &hosts, client.MatchingLabels{
		"digest.auto-discovery.experimental.securecodebox.io": podDigest[0:min(len(podDigest), 63)],
	})

	log.Info("Got Hosts for Service & Pods", "hosts", len(hosts.Items), "digest", podDigest)

	if len(hosts.Items) != 0 {
		r.Log.Info("Service Version was already scanned. Skipping.")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	// No scan for this pod digest yet. Scanning now

	host := targetsv1.Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service-%s", service.Name, podDigest[:min(len(podDigest), 10)]),
			Namespace: service.Namespace,
			Labels: map[string]string{
				"digest.auto-discovery.experimental.securecodebox.io": podDigest[0:min(len(podDigest), 63)],
			},
		},
		Spec: targetsv1.HostSpec{
			Hostname: fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace),
			Ports:    getHostPorts(service),
		},
	}

	err := r.Create(ctx, &host)
	if err != nil {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, err
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 5 * time.Second,
	}, nil
}

func getHostPorts(service corev1.Service) []targetsv1.HostPort {
	servicePorts := getLikelyHTTPPorts(service)

	httpIshPorts := []targetsv1.HostPort{}

	for _, port := range servicePorts {
		if port.Port == 443 || port.Port == 8443 || port.Name == "https" {
			httpIshPorts = append(httpIshPorts, targetsv1.HostPort{
				Port: port.Port,
				Type: "https",
			})
		} else {
			httpIshPorts = append(httpIshPorts, targetsv1.HostPort{
				Port: port.Port,
				Type: "http",
			})
		}
	}

	return httpIshPorts
}

func getLikelyHTTPPorts(service corev1.Service) []corev1.ServicePort {
	httpIshPorts := []corev1.ServicePort{}

	for _, port := range service.Spec.Ports {
		if port.Port == 80 ||
			port.Port == 8080 ||
			port.Port == 443 ||
			port.Port == 8443 ||
			// Node.js
			port.Port == 3000 ||
			// Flask
			port.Port == 5000 ||
			// Django
			port.Port == 8000 ||
			port.Name == "http" ||
			port.Name == "https" {
			httpIshPorts = append(httpIshPorts, port)
		}
	}

	return httpIshPorts
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func getShaHashForPod(pod corev1.Pod) *string {
	if len(pod.Status.ContainerStatuses) == 0 {
		return nil
	}

	containerStatus := pod.Status.ContainerStatuses[0]

	if containerStatus.ImageID == "" {
		return nil
	}

	var fullImageName string
	if strings.HasPrefix(containerStatus.ImageID, "docker-pullable://") {
		// Extract the fullImageName from the following format "docker-pullable://scbexperimental/parser-nmap@sha256:f953..."
		fullImageName = containerStatus.ImageID[18:]
	} else {
		return nil
	}

	imageSegments := strings.Split(fullImageName, "@")
	prefixedDigest := imageSegments[1]

	var truncatedDigest string
	if strings.HasPrefix(prefixedDigest, "sha256:") {
		// Only keep actual digest
		// Example from "sha256:f953bc6c5446c20ace8787a1956c2e46a2556cc7a37ef7fc0dda7b11dd87f73d"
		// What is kept: "f953bc6c5446c20ace8787a1956c2e46a2556cc7a37ef7fc0dda7b11dd87f73d"
		truncatedDigest = prefixedDigest[7:71]
		return &truncatedDigest
	}

	return nil
}

// SetupWithManager sets up the controller and initializes every thing it needs
func (r *ServiceScanReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(&targetsv1.Host{}, ".metadata.service-controller", func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		host := rawObj.(*targetsv1.Host)
		owner := metav1.GetControllerOf(host)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Service...
		if owner.APIVersion != "v1" || owner.Kind != "Service" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(getPredicates(mgr.GetClient(), r.Log)).
		Complete(r)
}
