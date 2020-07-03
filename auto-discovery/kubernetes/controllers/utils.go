package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func getNamespace(client client.Client, name string) (*corev1.Namespace, error) {
	namespace := corev1.Namespace{}
	err := client.Get(context.Background(), types.NamespacedName{Name: name}, &namespace)
	if err != nil {
		return nil, err
	}

	return &namespace, nil
}

func getPredicates(client client.Client, log logr.Logger) predicate.Predicate {

	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}

			namespace, err := getNamespace(client, event.Meta.GetNamespace())
			if err != nil {
				log.Error(err, "Failed to get Namespace")
			}

			if val, ok := namespace.GetAnnotations()["auto-discovery.experimental.securecodebox.io/enabled"]; ok && val == "true" {
				return true
			}
			return false
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}

			namespace, err := getNamespace(client, event.Meta.GetNamespace())
			if err != nil {
				log.Error(err, "Failed to get Namespace")
			}

			if val, ok := namespace.GetAnnotations()["auto-discovery.experimental.securecodebox.io/enabled"]; ok && val == "true" {
				return true
			}
			return false
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			if val, ok := event.MetaNew.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}

			namespace, err := getNamespace(client, event.MetaNew.GetNamespace())
			if err != nil {
				log.Error(err, "Failed to get Namespace")
			}

			if val, ok := namespace.GetAnnotations()["auto-discovery.experimental.securecodebox.io/enabled"]; ok && val == "true" {
				return true
			}
			return false
		},
		GenericFunc: func(event event.GenericEvent) bool {
			if val, ok := event.Meta.GetAnnotations()["auto-discovery.experimental.securecodebox.io/ignore"]; ok && val == "true" {
				return false
			}

			namespace, err := getNamespace(client, event.Meta.GetNamespace())
			if err != nil {
				log.Error(err, "Failed to get Namespace")
			}

			if val, ok := namespace.GetAnnotations()["auto-discovery.experimental.securecodebox.io/enabled"]; ok && val == "true" {
				return true
			}
			return false
		},
	}
}
