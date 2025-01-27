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
	"fmt"
	"os"
	"strings"

	cachev1alpha1 "github.com/example/minecraft-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// imageForMinecraft gets the Operand image which is managed by this controller
// from the MINECRAFT_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForMinecraft() (string, error) {
	var imageEnvVar = "MINECRAFT_IMAGE"
	var defaultImage = "itzg/minecraft-server:latest"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		log.Log.Info(fmt.Sprintf("MINECRAFT_IMAGE environment variable not found, using default image: %s", defaultImage))
		return defaultImage, nil
	}
	return image, nil
}

// labelsForMinecraft returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForMinecraft(minecraft *cachev1alpha1.Minecraft) map[string]string {
	var imageTag string
	image, err := imageForMinecraft()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	ls := make(map[string]string) // Initialize the map
	ls["app.kubernetes.io/name"] = "minecraft-operator"
	ls["app.kubernetes.io/version"] = imageTag
	ls["app.kubernetes.io/managed-by"] = "MinecraftController"
	ls["cache.example.com/name"] = minecraft.Name
	ls["containertype"] = "minecraft-server"

	return ls
}

// SetupWithManager sets up the controller with the Manager.
// Note that the Statefulset will be also watched in order to ensure its
// desirable state on the cluster
func (r *MinecraftReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Minecraft{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
