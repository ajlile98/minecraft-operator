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

func (r *MinecraftReconciler) createMinecraftService(
	ctx context.Context, minecraft *cachev1alpha1.Minecraft) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Define a new service
	svc, err := r.serviceForMinecraft(minecraft)
	if err != nil {
		log.Error(err, "Failed to define new Service resource for Minecraft")

		// The following implementation will update the status
		meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to create Service for the custom resource (%s): (%s)", minecraft.Name, err)})
		if err := r.Status().Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update Minecraft status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	log.Info("Creating a new Service",
		"Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
	if err = r.Create(ctx, svc); err != nil {
		log.Error(err, "Failed to create new Service",
			"Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		return ctrl.Result{}, err
	}

	// Service created successfully
	// We will requeue the reconciliation so that we can ensure the state
	// and move forward for the next operations
	return ctrl.Result{Requeue: true}, nil
}

// serviceForMinecraft returns a Minecraft Service object
func (r *MinecraftReconciler) serviceForMinecraft(
	minecraft *cachev1alpha1.Minecraft) (*corev1.Service, error) {
	ls := labelsForMinecraft(minecraft)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minecraft.Name,
			Namespace: minecraft.Namespace,
			Annotations: map[string]string{
				"mc-router.itzg.me/externalServerName": minecraft.Name + ".andylile.com",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Name:     "minecraft",
					Protocol: corev1.ProtocolTCP,
					Port:     25565,
					// TargetPort: intstr.FromInt(25565),
				},
			},
		},
	}

	// Set the ownerRef for the Service
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(minecraft, service, r.Scheme); err != nil {
		return nil, err
	}
	return service, nil
}
