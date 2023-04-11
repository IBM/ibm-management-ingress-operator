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
	"fmt"

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
		klog.Errorf("Error setting controller reference on ServiceAccount: %v", err)
	}

	klog.Infof("Creating ServiceAccount %q for %q.", ServiceAccountName, ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failure constructing ServiceAccount for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Create required clusterRole
	klog.Infof("Creating ClusterRole: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	_, err = ingressRequest.CreateClusterRole(AppName, defaultRules)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failure constructing ClusterRole for %q: %v", ingressRequest.managementIngress.Name, err)
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

	klog.Infof("Creating ClusterRoleBingding: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	err = ingressRequest.CreateClusterRoleBinding(clusterRoleBinding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failure constructing ClusterRoleBinding for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedServiceAccount", "Successfully created service account %q", ServiceAccountName)

	return nil
}
