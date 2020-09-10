//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package utils

import (
	"reflect"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
)

// GetAnnotation returns the value of an annoation for a given key and true if the key was found
func GetAnnotation(key string, meta metav1.ObjectMeta) (string, bool) {
	for k, value := range meta.Annotations {
		if k == key {
			return value, true
		}
	}
	return "", false
}

func AsOwner(o *v1alpha1.ManagementIngress) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: v1alpha1.SchemeGroupVersion.String(),
		Kind:       "ManagementIngress",
		Name:       o.Name,
		UID:        o.UID,
		Controller: GetBool(true),
	}
}

func AreMapsSame(lhs, rhs map[string]string) bool {
	return reflect.DeepEqual(lhs, rhs)
}

func AreTolerationsSame(lhs, rhs []core.Toleration) bool {
	if len(lhs) != len(rhs) {
		return false
	}

	for _, lhsToleration := range lhs {
		if !containsToleration(lhsToleration, rhs) {
			return false
		}
	}

	return true

}

func containsToleration(toleration core.Toleration, tolerations []core.Toleration) bool {
	for _, t := range tolerations {
		if isTolerationSame(t, toleration) {
			return true
		}
	}

	return false
}

func isTolerationSame(lhs, rhs core.Toleration) bool {

	tolerationSecondsBool := false
	// check that both are either null or not null
	if (lhs.TolerationSeconds == nil) == (rhs.TolerationSeconds == nil) {
		if lhs.TolerationSeconds != nil {
			// only compare values (attempt to dereference) if pointers aren't nil
			tolerationSecondsBool = (*lhs.TolerationSeconds == *rhs.TolerationSeconds)
		} else {
			tolerationSecondsBool = true
		}
	}

	return (lhs.Key == rhs.Key) &&
		(lhs.Operator == rhs.Operator) &&
		(lhs.Value == rhs.Value) &&
		(lhs.Effect == rhs.Effect) &&
		tolerationSecondsBool
}

func AppendTolerations(lhsTolerations, rhsTolerations []core.Toleration) []core.Toleration {
	if lhsTolerations == nil {
		lhsTolerations = []core.Toleration{}
	}

	return append(lhsTolerations, rhsTolerations...)
}

func GetBool(value bool) *bool {
	b := value
	return &b
}

func GetInt32(value int32) *int32 {
	i := value
	return &i
}

func GetInt64(value int64) *int64 {
	i := value
	return &i
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func PodVolumeEquivalent(lhs, rhs []core.Volume) bool {

	if len(lhs) != len(rhs) {
		return false
	}

	lhsMap := make(map[string]core.Volume)
	rhsMap := make(map[string]core.Volume)

	for _, vol := range lhs {
		lhsMap[vol.Name] = vol
	}

	for _, vol := range rhs {
		rhsMap[vol.Name] = vol
	}

	for name, lhsVol := range lhsMap {
		if rhsVol, ok := rhsMap[name]; ok {
			if lhsVol.Secret != nil && rhsVol.Secret != nil {
				if lhsVol.Secret.SecretName != rhsVol.Secret.SecretName {
					return false
				}

				continue
			}
			if lhsVol.ConfigMap != nil && rhsVol.ConfigMap != nil {
				if lhsVol.ConfigMap.LocalObjectReference.Name != rhsVol.ConfigMap.LocalObjectReference.Name {
					return false
				}

				continue
			}
			if lhsVol.HostPath != nil && rhsVol.HostPath != nil {
				if lhsVol.HostPath.Path != rhsVol.HostPath.Path {
					return false
				}
				continue
			}

			return false
		} else {
			// if rhsMap doesn't have the same key has lhsMap
			return false
		}
	}

	return true
}

/**
EnvValueEqual - check if 2 EnvValues are equal or not
Notes:
- reflect.DeepEqual does not return expected results if the to-be-compared value is a pointer.
- needs to adjust with k8s.io/api/core/v#/types.go when the types are updated.
**/
func EnvValueEqual(env1, env2 []core.EnvVar) bool {
	var found bool
	if len(env1) != len(env2) {
		return false
	}
	for _, elem1 := range env1 {
		found = false
		for _, elem2 := range env2 {
			if elem1.Name == elem2.Name {
				if elem1.Value != elem2.Value {
					return false
				}
				if (elem1.ValueFrom != nil && elem2.ValueFrom == nil) ||
					(elem1.ValueFrom == nil && elem2.ValueFrom != nil) {
					return false
				}
				if elem1.ValueFrom != nil {
					found = EnvVarSourceEqual(*elem1.ValueFrom, *elem2.ValueFrom)
				} else {
					found = true
				}
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func EnvVarSourceEqual(esource1, esource2 core.EnvVarSource) bool {
	if (esource1.FieldRef != nil && esource2.FieldRef == nil) ||
		(esource1.FieldRef == nil && esource2.FieldRef != nil) ||
		(esource1.ResourceFieldRef != nil && esource2.ResourceFieldRef == nil) ||
		(esource1.ResourceFieldRef == nil && esource2.ResourceFieldRef != nil) ||
		(esource1.ConfigMapKeyRef != nil && esource2.ConfigMapKeyRef == nil) ||
		(esource1.ConfigMapKeyRef == nil && esource2.ConfigMapKeyRef != nil) ||
		(esource1.SecretKeyRef != nil && esource2.SecretKeyRef == nil) ||
		(esource1.SecretKeyRef == nil && esource2.SecretKeyRef != nil) {
		return false
	}
	var rval bool
	if esource1.FieldRef != nil {
		if rval = reflect.DeepEqual(*esource1.FieldRef, *esource2.FieldRef); !rval {
			return rval
		}
	}
	if esource1.ResourceFieldRef != nil {
		if rval = EnvVarResourceFieldSelectorEqual(*esource1.ResourceFieldRef, *esource2.ResourceFieldRef); !rval {
			return rval
		}
	}
	if esource1.ConfigMapKeyRef != nil {
		if rval = reflect.DeepEqual(*esource1.ConfigMapKeyRef, *esource2.ConfigMapKeyRef); !rval {
			return rval
		}
	}
	if esource1.SecretKeyRef != nil {
		if rval = reflect.DeepEqual(*esource1.SecretKeyRef, *esource2.SecretKeyRef); !rval {
			return rval
		}
	}
	return true
}

func EnvVarResourceFieldSelectorEqual(resource1, resource2 core.ResourceFieldSelector) bool {
	if (resource1.ContainerName == resource2.ContainerName) &&
		(resource1.Resource == resource2.Resource) &&
		(resource1.Divisor.Cmp(resource2.Divisor) == 0) {
		return true
	}
	return false
}

func AppendAnnotations(lhsAnnotation, rhsAnnotation map[string]string) map[string]string {
	for k, v := range rhsAnnotation {
		lhsAnnotation[k] = v
	}

	return lhsAnnotation
}

func IsDeploymentDifferent(current *apps.Deployment, desired *apps.Deployment) (*apps.Deployment, bool) {

	different := false

	if !AreMapsSame(current.Spec.Template.Spec.NodeSelector, desired.Spec.Template.Spec.NodeSelector) {
		current.Spec.Template.Spec.NodeSelector = desired.Spec.Template.Spec.NodeSelector
		different = true
	}

	if !AreTolerationsSame(current.Spec.Template.Spec.Tolerations, desired.Spec.Template.Spec.Tolerations) {
		current.Spec.Template.Spec.Tolerations = desired.Spec.Template.Spec.Tolerations
		different = true
	}

	if isImageDifference(&current.Spec.Template.Spec, &desired.Spec.Template.Spec) {
		podSpec := updateCurrentImages(&current.Spec.Template.Spec, &desired.Spec.Template.Spec)
		current.Spec.Template.Spec = *podSpec
		different = true
	}

	if &current.Spec.Replicas != &desired.Spec.Replicas {
		current.Spec.Replicas = desired.Spec.Replicas
		different = true
	}

	if AreResourcesDifferent(current, desired) {
		different = true
	}

	if !EnvValueEqual(current.Spec.Template.Spec.Containers[0].Env, desired.Spec.Template.Spec.Containers[0].Env) {
		current.Spec.Template.Spec.Containers[0].Env = desired.Spec.Template.Spec.Containers[0].Env
		different = true
	}
	if !PodVolumeEquivalent(current.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		current.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
		different = true
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.Containers[0].VolumeMounts, desired.Spec.Template.Spec.Containers[0].VolumeMounts) {
		current.Spec.Template.Spec.Containers[0].VolumeMounts = desired.Spec.Template.Spec.Containers[0].VolumeMounts
		different = true
	}

	return current, different
}

func IsDaemonsetDifferent(current *apps.DaemonSet, desired *apps.DaemonSet) (*apps.DaemonSet, bool) {

	different := false

	if !AreMapsSame(current.Spec.Template.Spec.NodeSelector, desired.Spec.Template.Spec.NodeSelector) {
		current.Spec.Template.Spec.NodeSelector = desired.Spec.Template.Spec.NodeSelector
		different = true
	}

	if !AreTolerationsSame(current.Spec.Template.Spec.Tolerations, desired.Spec.Template.Spec.Tolerations) {
		current.Spec.Template.Spec.Tolerations = desired.Spec.Template.Spec.Tolerations
		different = true
	}

	if isImageDifference(&current.Spec.Template.Spec, &desired.Spec.Template.Spec) {
		podSpec := updateCurrentImages(&current.Spec.Template.Spec, &desired.Spec.Template.Spec)
		current.Spec.Template.Spec = *podSpec
		different = true
	}

	if AreResourcesDifferent(current, desired) {
		different = true
	}

	if !EnvValueEqual(current.Spec.Template.Spec.Containers[0].Env, desired.Spec.Template.Spec.Containers[0].Env) {
		current.Spec.Template.Spec.Containers[0].Env = desired.Spec.Template.Spec.Containers[0].Env
		different = true
	}
	if !PodVolumeEquivalent(current.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		current.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
		different = true
	}
	if !reflect.DeepEqual(current.Spec.Template.Spec.Containers[0].VolumeMounts, desired.Spec.Template.Spec.Containers[0].VolumeMounts) {
		current.Spec.Template.Spec.Containers[0].VolumeMounts = desired.Spec.Template.Spec.Containers[0].VolumeMounts
		different = true
	}

	return current, different
}

func isImageDifference(current *core.PodSpec, desired *core.PodSpec) bool {
	for _, curr := range current.Containers {
		for _, des := range desired.Containers {
			// Only compare the images of containers with the same name
			if curr.Name == des.Name {
				if curr.Image != des.Image {
					return true
				}
			}
		}
	}

	return false
}

func updateCurrentImages(current *core.PodSpec, desired *core.PodSpec) *core.PodSpec {
	containers := current.Containers

	for index, curr := range current.Containers {
		for _, des := range desired.Containers {
			// Only compare the images of containers with the same name
			if curr.Name == des.Name {
				if curr.Image != des.Image {
					containers[index].Image = des.Image
				}
			}
		}
	}

	return current
}
