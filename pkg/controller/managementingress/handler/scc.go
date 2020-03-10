package handler

import (
	"fmt"
	"strings"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
	scc "github.com/openshift/api/security/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

//NewSecurityContextConstraint stubs an instance of a SecurityContextConstraint
func NewSecurityContextConstraint(serviceaccount, name, namespace string) *scc.SecurityContextConstraints {
	user := strings.Join([]string{"system:serviceaccount", name, namespace}, ":")
	privilegeEscalation := false
	var priority int32 = 10

	labels := GetCommonLabels()

	return &scc.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecurityContextConstraint",
			APIVersion: scc.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Priority:                        &priority,
		AllowPrivilegedContainer:        false,
		DefaultAddCapabilities:          []core.Capability{},
		RequiredDropCapabilities:        []core.Capability{},
		AllowedCapabilities:             []core.Capability{},
		AllowHostDirVolumePlugin:        false,
		AllowedFlexVolumes:              []scc.AllowedFlexVolume{},
		AllowHostNetwork:                false,
		AllowHostPID:                    false,
		AllowHostIPC:                    false,
		DefaultAllowPrivilegeEscalation: &privilegeEscalation,
		AllowPrivilegeEscalation:        &privilegeEscalation,
		SELinuxContext:                  scc.SELinuxContextStrategyOptions{Type: scc.SELinuxStrategyMustRunAs},
		RunAsUser:                       scc.RunAsUserStrategyOptions{Type: scc.RunAsUserStrategyRunAsAny},
		SupplementalGroups:              scc.SupplementalGroupsStrategyOptions{Type: scc.SupplementalGroupsStrategyRunAsAny},
		FSGroup:                         scc.FSGroupStrategyOptions{Type: scc.FSGroupStrategyRunAsAny},
		ReadOnlyRootFilesystem:          false,
		Users:                           []string{user},
		Groups:                          []string{},
		SeccompProfiles:                 []string{"docker/default"},
		AllowedUnsafeSysctls:            []string{},
		ForbiddenSysctls:                []string{},
		Volumes:                         []scc.FSType{scc.FSTypeConfigMap, scc.FSTypeSecret},
	}
}

func (ingressRequest *IngressRequest) CreateSecurityContextConstraint() error {
	scc := NewSecurityContextConstraint(
		ServiceAccountName,
		SCCName,
		ingressRequest.managementIngress.Namespace,
	)

	utils.AddOwnerRefToObject(scc, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Creating SecurityContextConstraint %q for %q.", SCCName, ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(scc)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing SecurityContextConstraint for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedSecurityContextConstraint", "Successfully created SecurityContextConstraint %q", SCCName)

	return nil
}

//RemoveSecurityContextConstraint with given name
func (ingressRequest *IngressRequest) RemoveSecurityContextConstraint(name string) error {

	scc := &scc.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecurityContextConstraint",
			APIVersion: scc.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	klog.Infof("Removing SecurityContextConstraint for %q.", ingressRequest.managementIngress.Name)
	err := ingressRequest.Delete(scc)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %v SecurityContextConstraint %v", name, err)
	}

	return nil
}
