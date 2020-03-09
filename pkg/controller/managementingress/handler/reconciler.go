package handler

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
	"k8s.io/client-go/tools/record"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

func Reconcile(requestIngress *v1alpha1.ManagementIngress, requestClient client.Client, recorder record.EventRecorder) (err error) {
	ingressRequest := IngressRequest{
		client:            requestClient,
		managementIngress: requestIngress,
		recorder:          recorder,
	}

	// First time in reconcile set route host in status.
	if len(requestIngress.Status.Host) <= 0 {
		// Get route host
		status := &v1alpha1.ManagementIngressStatus{}
		host, err := getRouteHost(&ingressRequest)
		if err != nil {
			return err
		} else {
			status = &v1alpha1.ManagementIngressStatus{
				Conditions: map[string]v1alpha1.ConditionList{},
				PodState:   v1alpha1.PodStateMap{},
				Host:       host,
				State: v1alpha1.OperandState{
					Message: "Get router host for management ingress.",
					Status:  v1alpha1.StatusDeploying,
				},
			}
		}

		// Update CR status
		requestIngress.Status = *status
		if err := ingressRequest.UpdateStatus(requestIngress); err != nil {
			return err
		}
	}

	// Reconcile cert
	if err = ingressRequest.CreateOrUpdateCertificates(); err != nil {
		return fmt.Errorf("Unable to create or update certificates for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// create serviceAccount
	if err = ingressRequest.CreateServiceAccount(); err != nil {
		return fmt.Errorf("Unable to create serviceAccount for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// create scc
	if err = ingressRequest.CreateSecurityContextConstraint(); err != nil {
		return fmt.Errorf("Unable to create SecurityContextConstraint for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile service
	if err = ingressRequest.CreateOrUpdateService(); err != nil {
		return fmt.Errorf("Unable to create or update service for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile configmap
	if err = ingressRequest.CreateOrUpdateConfigMap(); err != nil {
		return fmt.Errorf("Unable to create or update configmap for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile route
	if err = ingressRequest.CreateOrUpdateRoute(); err != nil {
		return fmt.Errorf("Unable to create or update route for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Reconcile deployment
	if err = ingressRequest.CreateOrUpdateDeployment(); err != nil {
		return fmt.Errorf("Unable to create or update deployment for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}

func getRouteHost(ing *IngressRequest) (string, error) {

	if specifiedRouteHost := ing.managementIngress.Spec.RouteHost; len(specifiedRouteHost) > 0 {
		return specifiedRouteHost, nil
	}

	// User did not specify route host. Get the default one.
	appDomain, err := ing.GetRouteAppDomain()
	if err != nil {
		return "", err
	}

	return strings.Join([]string{RouteName, appDomain}, "."), nil
}
