/*
Copyright 2020 iteratec GmbH.

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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	executionv1 "github.com/secureCodeBox/secureCodeBox-v2-alpha/operator/apis/execution/v1"
	targetsv1 "github.com/secureCodeBox/secureCodeBox-v2-alpha/operator/apis/targets/v1"
)

// ContainerImageReconciler reconciles a ContainerImage object
type ContainerImageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type dockerConfigFile struct {
	Auths map[string]dockerAuthConfig `json:"auths"`
}

type dockerAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

// +kubebuilder:rbac:groups=targets.experimental.securecodebox.io,resources=containerimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=targets.experimental.securecodebox.io,resources=containerimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=execution.experimental.securecodebox.io,resources=scheduledscans,verbs=get;list;create
// +kubebuilder:rbac:groups=execution.experimental.securecodebox.io,resources=scheduledscans/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;create;patch;update

// Reconcile compares the scan object against the state of the cluster and updates both if needed
func (r *ContainerImageReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerImage", req.NamespacedName)

	var containerImage targetsv1.ContainerImage
	if err := r.Get(ctx, req.NamespacedName, &containerImage); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		log.V(7).Info("Unable to fetch ContainerImage")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var envVars []corev1.EnvVar

	if containerImage.Spec.ImagePullSecret != nil {
		// Need to get the ImagePullSecret and parse the .dockerconfig file...

		pullSecret := corev1.Secret{}

		if err := r.Get(ctx, types.NamespacedName{Name: containerImage.Spec.ImagePullSecret.Name, Namespace: req.Namespace}, &pullSecret); err != nil {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.V(7).Info("Unable to fetch PullSecret For ContainerImage Target")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		if pullSecret.Type != corev1.SecretTypeDockerConfigJson {
			return ctrl.Result{}, fmt.Errorf("Unsupported image pull secret type '%s'. Supported is '%s'", pullSecret.Type, corev1.SecretTypeDockerConfigJson)
		}

		if jsonBytes, ok := pullSecret.Data[corev1.DockerConfigJsonKey]; ok == true {
			log.Info("Found required key in imagePullSecret")

			var configFile dockerConfigFile
			if err := json.Unmarshal(jsonBytes, &configFile); err != nil {
				log.Error(err, "Failed to read Dockerconfig json file")
				return ctrl.Result{}, err
			}

			readableSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      truncateName(fmt.Sprintf("%s-readable", pullSecret.Name)),
					Namespace: containerImage.Namespace,
				},
			}

			for registry, auth := range configFile.Auths {
				if strings.HasPrefix(containerImage.Spec.Image, registry) {
					log.Info("Found the right registry", "registry", registry)

					if err := ctrl.SetControllerReference(&pullSecret, &readableSecret, r.Scheme); err != nil {
						log.Error(err, "Failed to set owner reference for readable secret")
						return ctrl.Result{}, err
					}

					_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &readableSecret, func() error {
						readableSecret.Data = map[string][]byte{
							"authURL":  []byte(registry),
							"username": []byte(auth.Username),
							"password": []byte(auth.Password),
						}

						return nil
					})
					if err != nil {
						log.Error(err, "Failed to create Readable Secret for Image Scan")
					}

					envVars = []corev1.EnvVar{
						{
							Name: "TRIVY_AUTH_URL",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-readable", pullSecret.Name),
									},
									Key: "authURL",
								},
							},
						},
						{
							Name: "TRIVY_USERNAME",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-readable", pullSecret.Name),
									},
									Key: "username",
								},
							},
						},
						{
							Name: "TRIVY_PASSWORD",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-readable", pullSecret.Name),
									},
									Key: "password",
								},
							},
						},
					}
					break
				}
			}
			log.Info("Done all successfully. Crazy right?!")
		}
	}

	var scheduledScan executionv1.ScheduledScan

	err := r.Get(ctx, types.NamespacedName{Name: containerImage.Name, Namespace: req.Namespace}, &scheduledScan)
	if apierrors.IsNotFound(err) {
		// Scan doesn't exist yet, creating now
		scheduledScan = executionv1.ScheduledScan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerImage.Name,
				Namespace: req.Namespace,
			},
			Spec: executionv1.ScheduledScanSpec{
				Interval:     metav1.Duration{Duration: 24 * time.Hour},
				HistoryLimit: 1,
				ScanSpec: &executionv1.ScanSpec{
					ScanType:   "trivy",
					Parameters: []string{containerImage.Spec.Image},
					Env:        envVars,
				},
			},
		}

		if err := ctrl.SetControllerReference(&containerImage, &scheduledScan, r.Scheme); err != nil {
			log.Error(err, "unable to set owner reference on ScheduledScan")
			return ctrl.Result{}, err
		}

		createErr := r.Create(ctx, &scheduledScan)
		if createErr != nil {
			r.Log.Error(createErr, "Failed to create Scan for ContainerImage")
			return ctrl.Result{}, createErr
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Updating the Targets Findings Status when the results have changed
	if !reflect.DeepEqual(containerImage.Status.Findings, scheduledScan.Status.Findings) {
		containerImage.Status.Findings = *scheduledScan.Status.Findings.DeepCopy()
		if err := r.Status().Update(ctx, &containerImage); err != nil {
			log.Error(err, "unable to update ContainerImage status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func truncateName(name string) string {
	if len(name) >= 55 {
		name = name[0:55]
	}

	// Ensure that the string does not end in a dot.
	// This would not be a valid domain name thous rejected by kubernetes
	if strings.HasSuffix(name, ".") {
		name = name[0:(len(name) - 1)]
	}

	return name
}

// SetupWithManager sets up the controller and initializes every thing it needs
func (r *ContainerImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&executionv1.ScheduledScan{}, ".metadata.containerImageController", func(rawObj runtime.Object) []string {
		// grab the scan object, extract the owner...
		scheduledScan := rawObj.(*executionv1.ScheduledScan)
		owner := metav1.GetControllerOf(scheduledScan)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Scan belonging to a ContainerImage...
		if owner.APIVersion != apiGVStr || owner.Kind != "ContainerImage" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&targetsv1.ContainerImage{}).
		Owns(&executionv1.ScheduledScan{}).
		Complete(r)
}
