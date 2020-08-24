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
package handler

import (
	"fmt"
	"os"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewServiceAccount(name string, namespace string) *core.ServiceAccount {

	labels := GetCommonLabels()

	return &core.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func (ingressRequest *IngressRequest) CreateServiceAccount() error {
	sa := NewServiceAccount(
		ServiceAccountName,
		ingressRequest.managementIngress.Namespace)

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, sa, ingressRequest.scheme); err != nil {
		klog.Error("Error setting controller reference on ServiceAccount: %v", err)
	}

	klog.Infof("Creating ServiceAccount %q for %q.", ServiceAccountName, ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing ServiceAccount for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		klog.Errorf("Failure getting watch namespace: %v", err)
		os.Exit(1)
	}

	// Create required clusterRole
	clusterrules := NewPolicyRules(
		NewPolicyRule(
			[]string{""},
			[]string{"nodes"},
			nil,
			[]string{"list", "watch"},
		),
		NewPolicyRule(
			[]string{"security.openshift.io"},
			[]string{"securitycontextconstraints"},
			[]string{SCCName},
			[]string{"use"},
		),
	)

	// Create required role
	rules := NewPolicyRules(
		NewPolicyRule(
			[]string{""},
			[]string{"services"},
			nil,
			[]string{"get", "list", "watch"},
		),
		NewPolicyRule(
			[]string{""},
			[]string{"endpoints", "pods", "secrets"},
			nil,
			[]string{"list", "watch"},
		),
		NewPolicyRule(
			[]string{""},
			[]string{"configmaps"},
			nil,
			[]string{"create", "get", "list", "update", "watch"},
		),
		NewPolicyRule(
			[]string{""},
			[]string{"events"},
			nil,
			[]string{"create", "patch"},
		),
		NewPolicyRule(
			[]string{"extensions", "networking.k8s.io"},
			[]string{"ingresses"},
			nil,
			[]string{"get", "list", "watch"},
		),
		NewPolicyRule(
			[]string{"extensions", "networking.k8s.io"},
			[]string{"ingresses/status"},
			nil,
			[]string{"update"},
		),
	)

	if len(namespace) == 0 {
		clusterrules = append(clusterrules, rules...)
	}

	klog.Infof("Creating ClusterRole: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	_, err = ingressRequest.CreateClusterRole(AppName, defaultRules)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing ClusterRole for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Create required clusterRoleBinding
	subject := rbac.Subject{
		Kind:      "ServiceAccount",
		Name:      ServiceAccountName,
		Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
	}
	clusterRoleBinding := NewClusterRoleBinding(
		AppName,
		AppName,
		NewSubjects(
			subject,
		),
	)

	klog.Infof("Creating ClusterRoleBinding: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	err = ingressRequest.CreateClusterRoleBinding(clusterRoleBinding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing ClusterRoleBinding for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	if len(namespace) > 0 {
		klog.Infof("Creating Role: %q for %q.", AppName, ingressRequest.managementIngress.Name)
		_, err = ingressRequest.CreateRole(AppName, ingressRequest.managementIngress.ObjectMeta.Namespace, rules)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing Role for %q: %v", ingressRequest.managementIngress.Name, err)
		}
	
		// Create required clusterRoleBinding
		subject := rbac.Subject{
			Kind:      "ServiceAccount",
			Name:      ServiceAccountName,
			Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
		}
		RoleBinding := NewRoleBinding(
			AppName,
			ingressRequest.managementIngress.ObjectMeta.Namespace,
			AppName,
			NewSubjects(
				subject,
			),
		)
	
		klog.Infof("Creating RoleBinding: %q for %q.", AppName, ingressRequest.managementIngress.Name)
		err = ingressRequest.CreateRoleBinding(RoleBinding)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing RoleBinding for %q: %v", ingressRequest.managementIngress.Name, err)
		}
	}

	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedServiceAccount", "Successfully created service account %q", ServiceAccountName)

	return nil
}
