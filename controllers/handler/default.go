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
package handler

import (
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultMemoryRequest resource.Quantity = resource.MustParse("300Mi")
	defaultCPURequest    resource.Quantity = resource.MustParse("50m")

	defaultMemoryLimit resource.Quantity = resource.MustParse("512Mi")
	defaultCPULimit    resource.Quantity = resource.MustParse("200m")
)

var defaultRules = []rbac.PolicyRule{
	{
		APIGroups:     []string{""},
		Resources:     []string{"services"},
		ResourceNames: nil,
		Verbs:         []string{"get", "list", "watch"},
	},
	{
		APIGroups:     []string{""},
		Resources:     []string{"endpoints", "nodes", "pods", "secrets"},
		ResourceNames: nil,
		Verbs:         []string{"list", "watch"},
	},
	{
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: nil,
		Verbs:         []string{"create", "get", "list", "update", "watch"},
	},
	{
		APIGroups:     []string{""},
		Resources:     []string{"events"},
		ResourceNames: nil,
		Verbs:         []string{"create", "patch"},
	},
	{
		APIGroups:     []string{"extensions", "networking.k8s.io"},
		Resources:     []string{"ingresses"},
		ResourceNames: nil,
		Verbs:         []string{"get", "list", "watch"},
	},
	{
		APIGroups:     []string{"extensions", "networking.k8s.io"},
		Resources:     []string{"ingresses/status"},
		ResourceNames: nil,
		Verbs:         []string{"update"},
	},
	{
		APIGroups:     []string{"security.openshift.io"},
		Resources:     []string{"securitycontextconstraints"},
		ResourceNames: []string{SCCName},
		Verbs:         []string{"use"},
	},
}
