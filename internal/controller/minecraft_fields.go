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

// The CRD API defines that the Minecraft type have a MinecraftSpec.Size field
// to set the quantity of Statefulset instances to the desired state on the cluster.
// Therefore, the following code will ensure the Statefulset size is the same as defined
// via the Size spec of the Custom Resource which we are reconciling.
// func (r *MinecraftReconciler) updateMinecraftSizeField(
// 	ctx context.Context, req ctrl.Request, minecraft *cachev1alpha1.Minecraft, statefulset *appsv1.StatefulSet) (ctrl.Result, error) {
// 	log := log.FromContext(ctx)

// 	size := minecraft.Spec.Size
// 	if *statefulset.Spec.Replicas != size {
// 		statefulset.Spec.Replicas = &size
// 		if err := r.Update(ctx, statefulset); err != nil {
// 			log.Error(err, "Failed to update Statefulset",
// 				"Statefulset.Namespace", statefulset.Namespace, "Statefulset.Name", statefulset.Name)

// 			// Re-fetch the minecraft Custom Resource before updating the status
// 			// so that we have the latest state of the resource on the cluster and we will avoid
// 			// raising the error "the object has been modified, please apply
// 			// your changes to the latest version and try again" which would re-trigger the reconciliation
// 			if err := r.Get(ctx, req.NamespacedName, minecraft); err != nil {
// 				log.Error(err, "Failed to re-fetch minecraft")
// 				return ctrl.Result{}, err
// 			}

// 			// The following implementation will update the status
// 			meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
// 				Status: metav1.ConditionFalse, Reason: "Resizing",
// 				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", minecraft.Name, err)})

// 			if err := r.Status().Update(ctx, minecraft); err != nil {
// 				log.Error(err, "Failed to update Minecraft status")
// 				return ctrl.Result{}, err
// 			}

// 			return ctrl.Result{}, err
// 		}

// 		// Now, that we update the size we want to requeue the reconciliation
// 		// so that we can ensure that we have the latest state of the resource before
// 		// update. Also, it will help ensure the desired state on the cluster
// 		return ctrl.Result{Requeue: true}, nil
// 	}

// 	// The following implementation will update the status
// 	meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
// 		Status: metav1.ConditionTrue, Reason: "Reconciling",
// 		Message: fmt.Sprintf("Statefulset for custom resource (%s) with %d replicas created successfully", minecraft.Name, size)})

// 	if err := r.Status().Update(ctx, minecraft); err != nil {
// 		log.Error(err, "Failed to update Minecraft status")
// 		return ctrl.Result{}, err
// 	}
// }
