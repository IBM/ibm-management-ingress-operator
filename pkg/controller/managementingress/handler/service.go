package handler

import (
	"fmt"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

//NewService stubs an instance of a Service
func NewService(name string, namespace string, servicePorts []core.ServicePort) *core.Service {

	labels := GetCommonLabels()

	return &core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"component": AppName,
			},
			Ports: servicePorts,
		},
	}
}

func (ingressRequest *IngressRequest) CreateOrUpdateService() error {
	service := NewService(
		ServiceName,
		ingressRequest.managementIngress.Namespace,
		[]core.ServicePort{
			{
				Port: 443,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
			},
		})

	utils.AddOwnerRefToObject(service, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Creating Service %q for %q.", ServiceName, ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(service)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing service for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedService", "Successfully created service %q", ServiceName)

	return nil
}

//RemoveService with given name and namespace
func (ingressRequest *IngressRequest) RemoveService(name string) error {

	service := &core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.Namespace,
		},
		Spec: core.ServiceSpec{},
	}

	klog.Infof("Removing Service for %q.", ingressRequest.managementIngress.Name)
	err := ingressRequest.Delete(service)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %v service %v", name, err)
	}

	return nil
}
