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
	"time"

	cachev1alpha1 "github.com/example/minecraft-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *MinecraftReconciler) createMinecraftPVC(
	ctx context.Context, minecraft *cachev1alpha1.Minecraft) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// define new pvc
	pvc, err := r.pvcForMinecraft(minecraft)
	if err != nil {
		log.Error(err, "Failed to define new PVC resource for Minecraft")

		// the following will update minecraft resource status
		meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to create PVC for the custom resource (%s): (%s)", minecraft.Name, err)})

		if err := r.Status().Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update Minecraft status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	log.Info("Creating a new PVC",
		"PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
	if err = r.Create(ctx, pvc); err != nil {
		log.Error(err, "failed to create new PVC",
			"PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil

}

func (r *MinecraftReconciler) pvcForMinecraft(
	minecraft *cachev1alpha1.Minecraft) (*corev1.PersistentVolumeClaim, error) {
	ls := labelsForMinecraft(minecraft)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minecraft.Name,
			Namespace: minecraft.Namespace,
			Labels:    ls,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(minecraft, pvc, r.Scheme); err != nil {
		return nil, err
	}

	return pvc, nil
}
