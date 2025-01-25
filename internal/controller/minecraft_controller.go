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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/example/minecraft-operator/api/v1alpha1"
)

const minecraftFinalizer = "cache.example.com/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailableMinecraft represents the status of the Statefulset reconciliation
	typeAvailableMinecraft = "Available"
	// typeDegradedMinecraft represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	typeDegradedMinecraft = "Degraded"
)

// MinecraftReconciler reconciles a Minecraft object
type MinecraftReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions (RBAC) on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

// +kubebuilder:rbac:groups=cache.example.com,resources=minecrafts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=minecrafts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=minecrafts/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=minecrafts.cache.example.com,resources=minecrafts,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *MinecraftReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Minecraft instance
	// The purpose is check if the Custom Resource for the Kind Minecraft
	// is applied on the cluster if not we return nil to stop the reconciliation
	minecraft := &cachev1alpha1.Minecraft{}
	err := r.Get(ctx, req.NamespacedName, minecraft)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("minecraft resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get minecraft")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if minecraft.Status.Conditions == nil || len(minecraft.Status.Conditions) == 0 {
		meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update Minecraft status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the minecraft Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, minecraft); err != nil {
			log.Error(err, "Failed to re-fetch minecraft")
			return ctrl.Result{}, err
		}
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource is deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(minecraft, minecraftFinalizer) {
		log.Info("Adding Finalizer for Minecraft")
		if ok := controllerutil.AddFinalizer(minecraft, minecraftFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the Minecraft instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMinecraftMarkedToBeDeleted := minecraft.GetDeletionTimestamp() != nil
	if isMinecraftMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(minecraft, minecraftFinalizer) {
			log.Info("Performing Finalizer Operations for Minecraft before delete CR")

			// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
			meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeDegradedMinecraft,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", minecraft.Name)})

			if err := r.Status().Update(ctx, minecraft); err != nil {
				log.Error(err, "Failed to update Minecraft status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before removing the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsForMinecraft(minecraft)

			// TODO(user): If you add operations to the doFinalizerOperationsForMinecraft method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the minecraft Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, minecraft); err != nil {
				log.Error(err, "Failed to re-fetch minecraft")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeDegradedMinecraft,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", minecraft.Name)})

			if err := r.Status().Update(ctx, minecraft); err != nil {
				log.Error(err, "Failed to update Minecraft status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for Minecraft after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(minecraft, minecraftFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for Minecraft")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, minecraft); err != nil {
				log.Error(err, "Failed to remove finalizer for Minecraft")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// check if pvc already exists, if not create a new one
	foundPvc := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: minecraft.Name, Namespace: minecraft.Namespace}, foundPvc)
	if err != nil && apierrors.IsNotFound(err) {
		return r.createMinecraftPVC(ctx, minecraft)
	} else if err != nil {
		log.Error(err, "Failed to get PVC")
		return ctrl.Result{}, err
	}

	configmap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: minecraft.Name, Namespace: minecraft.Namespace}, configmap)
	if err != nil && apierrors.IsNotFound(err) {
		return r.createMinecraftConfigMap(ctx, minecraft)
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		// return error for the reconciliation be retriggered
		return ctrl.Result{}, err
	}

	// Check if the statefulset already exists, if not create a new one
	statefulset := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: minecraft.Name, Namespace: minecraft.Namespace}, statefulset)
	if err != nil && apierrors.IsNotFound(err) {
		return r.createMinecraftStatefulSet(ctx, minecraft)
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// create a service for the Minecraft Statefulset
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: minecraft.Name, Namespace: minecraft.Namespace}, service)
	if err != nil && apierrors.IsNotFound(err) {
		return r.createMinecraftService(ctx, minecraft)
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// r.updateMinecraftSizeField(ctx, minecraft, statefulset)

	return ctrl.Result{}, nil
}

// finalizeMinecraft will perform the required operations before delete the CR.
func (r *MinecraftReconciler) doFinalizerOperationsForMinecraft(cr *cachev1alpha1.Minecraft) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of deleting resources which are
	// created and managed in the reconciliation. These ones, such as the Statefulset created on this reconcile,
	// are defined as dependent of the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Statefulset will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}
