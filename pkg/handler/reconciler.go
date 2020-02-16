package handler

import (
	"fmt"

	"k8s.io/client-go/tools/record"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	csv1alpha1 "github.com/IBM/management-ingress-operator/pkg/apis/cs/v1alpha1"
)

func Reconcile(requestIngress *csv1alpha1.ManagementIngress, requestClient client.Client, recorder record.EventRecorder) (err error) {
	ingressRequest := IngressRequest{
		client:            requestClient,
		managementIngress: requestIngress,
		recorder:          recorder,
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
