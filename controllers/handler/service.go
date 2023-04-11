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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/IBM/ibm-management-ingress-operator/utils"
)

// NewService stubs an instance of a Service
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
				Name:     "https",
				Port:     443,
				Protocol: core.ProtocolTCP,
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "https",
				},
			},
		})

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, service, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Service: %v", err)
	}

	err := ingressRequest.Create(service)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "CreateService", "Failed to create service %q", ServiceName)
			return fmt.Errorf("failure creating service: %v", err)
		}

		klog.Infof("Trying to update service: %s as it already existed.", ServiceName)
		current := &core.Service{}
		if err = ingressRequest.Get(ServiceName, ingressRequest.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("failure getting %q service for %q: %v", ServiceName, ingressRequest.managementIngress.Name, err)
		}

		desired, different := utils.IsServiceDifferent(current, service)
		if !different {
			klog.Infof("No change found from the service: %s, skip updating current service.", ServiceName)
			return nil
		}
		klog.Infof("Found change for service: %s, trying to update it.", ServiceName)
		err = ingressRequest.Update(desired)
		if err != nil {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdateService", "Failed to update service: %s", ServiceName)
			return fmt.Errorf("failure updating %q service for %q: %v", ServiceName, ingressRequest.managementIngress.Name, err)
		}

		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "UpdateService", "Successfully updated service %q", ServiceName)
		return nil
	}

	klog.Infof("Created Service: %q.", ServiceName)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreateService", "Successfully created service %q", ServiceName)

	return nil
}
