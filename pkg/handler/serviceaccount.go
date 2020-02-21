package handler

import (
	"fmt"

	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
)

func NewServiceAccount(name string, namespace string) *core.ServiceAccount {
	return &core.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func (ingressRequest *IngressRequest) CreateServiceAccount() error {
	sa := NewServiceAccount(
		ServiceAccountName,
		ingressRequest.managementIngress.Namespace)

	utils.AddOwnerRefToObject(sa, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Creating ServiceAccount %q for %q.", ServiceAccountName, ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing ServiceAccount for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Create required clusterRole
	clusterrules := NewPolicyRules(
		NewPolicyRule(
			[]string{""},
			[]string{"services"},
			nil,
			[]string{"get", "list", "watch"},
		),
		NewPolicyRule(
			[]string{""},
			[]string{"endpoints", "nodes", "pods", "secrets"},
			nil,
			[]string{"list", "watch"},
		),
		NewPolicyRule(
			[]string{""},
			[]string{"configmaps"},
			nil,
			[]string{"get", "list", "update", "watch"},
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

	klog.Infof("Creating ClusterRole: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	_, err = ingressRequest.CreateClusterRole(AppName, clusterrules)
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

	klog.Infof("Creating ClusterRoleBingding: %q for %q.", AppName, ingressRequest.managementIngress.Name)
	err = ingressRequest.CreateClusterRoleBinding(clusterRoleBinding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing ClusterRoleBinding for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedServiceAccount", "Successfully created service account %q", ServiceAccountName)

	return nil
}

//RemoveService with given name and namespace
func (ingressRequest *IngressRequest) RemoveServiceAccount(name string) error {

	klog.V(4).Infof("Deleting related ClusterRole for ManagementIngress %q.", ingressRequest.managementIngress.Name)
	clusterRole := &rbac.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbac.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AppName,
		},
		Rules: []rbac.PolicyRule{},
	}
	err := ingressRequest.Delete(clusterRole)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting ClusterRole: %v", err)
	}

	klog.V(4).Infof("Deleting related ClusterRoleBingding for ManagementIngress %q.", ingressRequest.managementIngress.Name)
	clusterRoleBinding := &rbac.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbac.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: AppName,
		},
		RoleRef:  rbac.RoleRef{},
		Subjects: []rbac.Subject{},
	}
	err = ingressRequest.Delete(clusterRoleBinding)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting ClusterRoleBinding: %v", err)
	}

	klog.Infof("Deleting SerrviceAccount: %q for ManagementIngress %q.", name, ingressRequest.managementIngress.Name)
	sa := &core.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.Namespace,
		},
	}

	err = ingressRequest.Delete(sa)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %v ServiceAccount %v", name, err)
	}

	klog.Infof("Successfully removed SerrviceAccount, ClusterRole, ClusterRoleBingding for ManagementIngress %q.", ingressRequest.managementIngress.Name)

	return nil
}
