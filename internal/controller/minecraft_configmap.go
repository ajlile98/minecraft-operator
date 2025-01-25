/*
Copyright 2024.

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
	"fmt"

	cachev1alpha1 "github.com/example/minecraft-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *MinecraftReconciler) createMinecraftConfigMap(
	ctx context.Context, minecraft *cachev1alpha1.Minecraft) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Define new configmap
	cm, err := r.configmapForMinecraft(minecraft)
	if err != nil {
		log.Error(err, "Failed to create new ConfigMap resource for Minecraft")

		// the following will update the status
		meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to creat ConfigMap for the custom resource (%s): (%s)", minecraft.Name, err)})

		if err := r.Status().Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update Minecraft Status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	log.Info("Creating a new ConfigMap",
		"ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
	if err = r.Create(ctx, cm); err != nil {
		log.Error(err, "Failed to create a new ConfigMap",
			"ConfigMap.Namespace", cm.Namespace, "Service.Name", cm.Name)
		return ctrl.Result{}, err
	}

	// ConfigMap created Successfully
	// Requeue reconciliation so that we can ensure the state
	// and move forward for the next operations
	return ctrl.Result{Requeue: true}, nil
}

func (r *MinecraftReconciler) configmapForMinecraft(
	minecraft *cachev1alpha1.Minecraft) (*corev1.ConfigMap, error) {
	ls := labelsForMinecraft(minecraft)
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minecraft.Name,
			Namespace: minecraft.Namespace,
			Labels:    ls,
		},
		Data: map[string]string{
			"EULA": "TRUE",
		},
	}

	if err := ctrl.SetControllerReference(minecraft, configmap, r.Scheme); err != nil {
		return nil, err
	}
	return configmap, nil
}
