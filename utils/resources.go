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
package utils

import (
	"reflect"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func CompareResources(current, desired v1.ResourceRequirements) (bool, v1.ResourceRequirements) {

	changed := false

	if !reflect.DeepEqual(desired.Limits, current.Limits) {
		changed = true
	}

	if !reflect.DeepEqual(desired.Requests, current.Requests) {
		changed = true
	}

	return changed, desired
}

func AreResourcesDifferent(current, desired interface{}) bool {

	var currentContainers []v1.Container
	var desiredContainers []v1.Container

	currentType := reflect.TypeOf(current)
	desiredType := reflect.TypeOf(desired)

	if currentType != desiredType {
		klog.Warningf("Attempting to compare resources for different types [%v] and [%v]", currentType, desiredType)
		return false
	}

	switch currentType {
	case reflect.TypeOf(&apps.Deployment{}):
		currentContainers = current.(*apps.Deployment).Spec.Template.Spec.Containers
		desiredContainers = desired.(*apps.Deployment).Spec.Template.Spec.Containers

	case reflect.TypeOf(&apps.DaemonSet{}):
		currentContainers = current.(*apps.DaemonSet).Spec.Template.Spec.Containers
		desiredContainers = desired.(*apps.DaemonSet).Spec.Template.Spec.Containers

	case reflect.TypeOf(&batch.CronJob{}):
		currentContainers = current.(*batch.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
		desiredContainers = desired.(*batch.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers

	default:
		klog.Warningf("Attempting to check resources for unmatched type [%v]", currentType)
		return false
	}

	containers := currentContainers
	changed := false

	for index, curr := range currentContainers {
		for _, des := range desiredContainers {
			// Only compare the images of containers with the same name
			if curr.Name == des.Name {
				if different, updated := CompareResources(curr.Resources, des.Resources); different {
					containers[index].Resources = updated
					changed = true
				}
			}
		}
	}

	return changed
}
