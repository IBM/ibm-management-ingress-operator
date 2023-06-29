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
package handler

import (
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/klog"

	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
)

func Reconcile(ingressRequest *IngressRequest, clusterType string, domainName string, loadBalanced string) (err error) {

	// First time in reconcile set route host in status.
	requestIngress := ingressRequest.managementIngress

	var host string
	if clusterType == CNCF {
		host = getHostOnCNCF(domainName)
	} else {
		// Get route host
		host, err = getRouteHost(ingressRequest)
		if err != nil {
			return err
		}
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
	fmt.Println("Reconciling cert")
	// Reconcile cert
	if err = ingressRequest.CreateOrUpdateCertificates(); err != nil {
		return fmt.Errorf("unable  to create or update certificates for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	fmt.Println("Reconciling service")
	// Reconcile service
	if err = ingressRequest.CreateOrUpdateService(loadBalanced); err != nil {
		return fmt.Errorf("unable  to create or update service for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	fmt.Println("Reconciling configmap")
	// Reconcile configmap
	if clusterType == CNCF {
		if err = ingressRequest.CreateOrUpdateConfigMap(clusterType, domainName); err != nil {
			return fmt.Errorf("unable  to create or update configmap for %q: %v", ingressRequest.managementIngress.Name, err)
		}
	} else {
		if err = ingressRequest.CreateOrUpdateConfigMap("", ""); err != nil {
			return fmt.Errorf("unable  to create or update configmap for %q: %v", ingressRequest.managementIngress.Name, err)
		}
	}

	// Reconcile route
	if clusterType != CNCF {
		fmt.Println("Reconciling route")
		// Reconcile route on ocp clusters
		if err = ingressRequest.CreateOrUpdateRoute(loadBalanced); err != nil {
			return fmt.Errorf("unable  to create or update route for %q: %v", ingressRequest.managementIngress.Name, err)
		}
	} else {
		// K cluster uses the same ca cert from the "route-tls-secret" secret as the ocp cluster
		// only create ibmcloud-cluster-ca-cert on cncf cluster, no route needed to be created
		stop := WaitForTimeout(10 * time.Minute)
		secret, err := waitForSecret(ingressRequest, RouteSecret, stop)
		if err != nil {
			return err
		}

		var caCert = secret.Data["ca.crt"]
		// Create or update secret ibmcloud-cluster-ca-cert
		if err := createClusterCACert(ingressRequest, ClusterSecretName, os.Getenv(PODNAMESPACE), caCert); err != nil {
			return fmt.Errorf("failure creating or updating secret: %v", err)
		}
	}

	// Reconcile deployment
	if err = ingressRequest.CreateOrUpdateDeployment(clusterType); err != nil {
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

	if ing.managementIngress.Spec.MultipleInstancesEnabled {
		multipleInstanceRouteName := strings.Join([]string{ConsoleRouteName, ing.managementIngress.Namespace}, "-")
		return strings.Join([]string{multipleInstanceRouteName, appDomain}, "."), nil
	}

	return strings.Join([]string{ConsoleRouteName, appDomain}, "."), nil
}

// Get the host for cncf env
func getHostOnCNCF(domainName string) string {
	return strings.Join([]string{ConsoleRouteName, domainName}, ".")
}
