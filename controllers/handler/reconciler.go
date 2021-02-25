//
// Copyright 2021 IBM Corporation
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
	"strings"
	"time"

	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
	"k8s.io/klog"
)

func Reconcile(ingressRequest *IngressRequest) (err error) {

	// First time in reconcile set route host in status.
	requestIngress := ingressRequest.managementIngress
	// Get route host
	host, err := getRouteHost(ingressRequest)
	if err != nil {
		return err
	}

	// see if Status.Host needs to be updated base on the routeHost value in the CR
	if len(requestIngress.Status.Host) == 0 || requestIngress.Status.Host != host {
		klog.Infof("Setting Status.Host to %s", host)
		status := &operatorv1alpha1.ManagementIngressStatus{
			Conditions: map[string]operatorv1alpha1.ConditionList{},
			PodState:   operatorv1alpha1.PodStateMap{},
			Host:       host,
			State: operatorv1alpha1.OperandState{
				Message: "Get router host for management ingress at " + time.Now().Format("2006-01-02 15:04:05"),
				Status:  operatorv1alpha1.StatusDeploying,
			},
		}

		// Update CR status
		requestIngress.Status = *status
		if err := ingressRequest.UpdateStatus(requestIngress); err != nil {
			return err
		}
	}

	// Reconcile cert
	if err = ingressRequest.CreateOrUpdateCertificates(); err != nil {
		return fmt.Errorf("unable  to create or update certificates for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile service
	if err = ingressRequest.CreateOrUpdateService(); err != nil {
		return fmt.Errorf("unable  to create or update service for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile configmap
	if err = ingressRequest.CreateOrUpdateConfigMap(); err != nil {
		return fmt.Errorf("unable  to create or update configmap for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile route
	if err = ingressRequest.CreateOrUpdateRoute(); err != nil {
		return fmt.Errorf("unable  to create or update route for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile deployment
	if err = ingressRequest.CreateOrUpdateDeployment(); err != nil {
		return fmt.Errorf("unable  to create or update deployment for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}

// Get the host for the cp-console route
func getRouteHost(ing *IngressRequest) (string, error) {

	if specifiedRouteHost := ing.managementIngress.Spec.RouteHost; len(specifiedRouteHost) > 0 {
		return specifiedRouteHost, nil
	}

	// User did not specify route host. Get the default one.
	appDomain, err := ing.GetRouteAppDomain()
	if err != nil {
		return "", err
	}

	return strings.Join([]string{ConsoleRouteName, appDomain}, "."), nil
}
