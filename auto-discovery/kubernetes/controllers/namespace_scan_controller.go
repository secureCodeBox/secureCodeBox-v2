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
	"time"

	"github.com/go-logr/logr"
	executionv1 "github.com/secureCodeBox/secureCodeBox-v2-alpha/operator/apis/execution/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceScanReconciler reconciles a Namespace object
type NamespaceScanReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking,resources=ingress,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking,resources=ingress/status,verbs=get

// Reconcile compares the Ingress object against the state of the cluster and updates both if needed
func (r *NamespaceScanReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log

	log.Info("Something happened to a namespace", "namespace", req.Name)

	scan := executionv1.ScheduledScan{}
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("kube-hunter-%s", req.Name), Namespace: req.Name}, &scan)
	if !apierrors.IsNotFound(err) {
		// Already exists, or errored
		return ctrl.Result{}, err
	}

	scan = executionv1.ScheduledScan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kube-hunter-%s", req.Name),
			Namespace: req.Name,
		},
		Spec: executionv1.ScheduledScanSpec{
			Interval:     metav1.Duration{Duration: 7 * 24 * time.Hour},
			HistoryLimit: 1,
			ScanSpec: &executionv1.ScanSpec{
				ScanType:   "kube-hunter",
				Parameters: []string{"--pod"},
			},
		},
	}

	err = r.Create(ctx, &scan)
	if err != nil {
		log.Error(err, "Failed to create kube-hunter scan")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller and initializes every thing it needs
func (r *NamespaceScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(getPredicates(mgr.GetClient(), r.Log)).
		Complete(r)
}
