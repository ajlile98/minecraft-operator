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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *MinecraftReconciler) createMinecraftStatefulSet(
	ctx context.Context, minecraft *cachev1alpha1.Minecraft) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// Define a new statefulset
	statefulset, err := r.statefulsetForMinecraft(minecraft)
	if err != nil {
		log.Error(err, "Failed to define new StatefulSet resource for Minecraft")

		// The following implementation will update the status
		meta.SetStatusCondition(&minecraft.Status.Conditions, metav1.Condition{Type: typeAvailableMinecraft,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to create StatefulSet for the custom resource (%s): (%s)", minecraft.Name, err)})

		if err := r.Status().Update(ctx, minecraft); err != nil {
			log.Error(err, "Failed to update Minecraft status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	log.Info("Creating a new StatefulSet",
		"StatefulSet.Namespace", statefulset.Namespace, "StatefulSet.Name", statefulset.Name)
	if err = r.Create(ctx, statefulset); err != nil {
		log.Error(err, "Failed to create new StatefulSet",
			"StatefulSet.Namespace", statefulset.Namespace, "StatefulSet.Name", statefulset.Name)
		return ctrl.Result{}, err
	}

	// StatefulSet created successfully
	// We will requeue the reconciliation so that we can ensure the state
	// and move forward for the next operations
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// statefulsetForMinecraft returns a Minecraft StatefulSet object
func (r *MinecraftReconciler) statefulsetForMinecraft(
	minecraft *cachev1alpha1.Minecraft) (*appsv1.StatefulSet, error) {
	ls := labelsForMinecraft(minecraft)
	replicas := int32(1) //minecraft.Spec.Size

	// Get the Operand image
	image, err := imageForMinecraft()
	if err != nil {
		return nil, err
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minecraft.Name,
			Namespace: minecraft.Namespace,
			Annotations: map[string]string{
				"reloader.stakater.com/auto": "true",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			ServiceName: minecraft.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					// TODO(user): Uncomment the following code to configure the nodeAffinity expression
					// according to the platforms which are supported by your solution. It is considered
					// best practice to support multiple architectures. build your manager image using the
					// makefile target docker-buildx. Also, you can use docker manifest inspect <image>
					// to check what are the platforms supported.
					// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
					//Affinity: &corev1.Affinity{
					//	NodeAffinity: &corev1.NodeAffinity{
					//		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					//			NodeSelectorTerms: []corev1.NodeSelectorTerm{
					//				{
					//					MatchExpressions: []corev1.NodeSelectorRequirement{
					//						{
					//							Key:      "kubernetes.io/arch",
					//							Operator: "In",
					//							Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
					//						},
					//						{
					//							Key:      "kubernetes.io/os",
					//							Operator: "In",
					//							Values:   []string{"linux"},
					//						},
					//					},
					//				},
					//			},
					//		},
					//	},
					//},

					// TODO: Uncomment the following code to configure the tolerations
					// SecurityContext: &corev1.PodSecurityContext{
					// 	RunAsNonRoot: &[]bool{true}[0],
					// 	// IMPORTANT: seccomProfile was introduced with Kubernetes 1.19
					// 	// If you are looking for to produce solutions to be supported
					// 	// on lower versions you must remove this option.
					// 	SeccompProfile: &corev1.SeccompProfile{
					// 		Type: corev1.SeccompProfileTypeRuntimeDefault,
					// 	},
					// },
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "minecraft",
						ImagePullPolicy: corev1.PullIfNotPresent,
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: minecraft.Name,
									},
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							InitialDelaySeconds: 90,
							PeriodSeconds:       15,
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"mc-monitor", "status"},
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: 30,
							PeriodSeconds:       5,
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"mc-monitor", "status"},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      minecraft.Name,
								MountPath: "/data",
							},
						},
						// TODO(user): Uncomment the following code to configure the resources
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						// SecurityContext: &corev1.SecurityContext{
						// 	RunAsNonRoot:             &[]bool{true}[0],
						// 	RunAsUser:                &[]int64{1000}[0],
						// 	AllowPrivilegeEscalation: &[]bool{false}[0],
						// 	Capabilities: &corev1.Capabilities{
						// 		Drop: []corev1.Capability{
						// 			"ALL",
						// 		},
						// 	},
						// },
					}},
					Volumes: []corev1.Volume{
						{
							Name: minecraft.Name,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: minecraft.Name,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set the ownerRef for the Statefulset
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(minecraft, statefulset, r.Scheme); err != nil {
		return nil, err
	}
	return statefulset, nil
}
