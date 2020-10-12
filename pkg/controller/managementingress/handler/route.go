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
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	operatorv1 "github.com/openshift/api/operator/v1"
	route "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//NewRoute stubs an instance of a Route
func NewRoute(name, namespace, serviceName, routeHost string, cert, key, caCert, destinationCAcert []byte) *route.Route {

	labels := GetCommonLabels()

	return &route.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: route.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: route.RouteSpec{
			Host: routeHost,
			To: route.RouteTargetReference{
				Name: serviceName,
				Kind: "Service",
			},
			TLS: &route.TLSConfig{
				Termination:                   route.TLSTerminationReencrypt,
				InsecureEdgeTerminationPolicy: route.InsecureEdgeTerminationPolicyRedirect,
				Certificate:                   string(cert),
				Key:                           string(key),
				CACertificate:                 string(caCert),
				DestinationCACertificate:      string(destinationCAcert),
			},
		},
	}
}

func NewSecret(name, namespace string, caCert []byte) *core.Secret {

	labels := GetCommonLabels()
	return &core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"ca.crt": caCert,
		},
	}
}

func (ingressRequest *IngressRequest) createOrUpdateSecret(secretName, namespace string, caCert []byte) error {
	// create ibmcloud-cluster-ca-cert
	clusterSecret := NewSecret(secretName, namespace, caCert)

	klog.Infof("create secret: %s for %q.", secretName, ingressRequest.managementIngress.ObjectMeta.Name)
	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, clusterSecret, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Secret: %v", err)
	}
	err := ingressRequest.Create(clusterSecret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure creating secret for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
		}

		klog.Infof("Trying to update Secret: %s for %q as it already existed.", secretName, ingressRequest.managementIngress.Name)
		// Update config
		current, err := ingressRequest.GetSecret(secretName)
		if err != nil {
			return fmt.Errorf("Failure getting Secret: %q  for %q: %v", secretName, ingressRequest.managementIngress.Name, err)
		}

		// no data change, just return
		if reflect.DeepEqual(clusterSecret.Data, current.Data) {
			klog.Infof("No change found from the Secret: %s.", secretName)
			return nil
		}

		json, _ := json.Marshal(clusterSecret)
		klog.Infof("Found change from Secret %s. Trying to update it.", json)
		current.Data = clusterSecret.Data

		// Apply the latest change to configmap
		if err = ingressRequest.Update(current); err != nil {
			return fmt.Errorf("Failure updating Secret: %v for %q: %v", secretName, ingressRequest.managementIngress.Name, err)
		}
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedSecret", "Successfully created secret %q", secretName)

	return nil
}

func (ingressRequest *IngressRequest) CreateOrUpdateRoute() error {
	// Get TLS secret for OCP route
	secret, err := ingressRequest.GetSecret(RouteSecret)
	if err != nil {
		return err
	}
	cert := secret.Data[core.TLSCertKey]
	key := secret.Data[core.TLSPrivateKeyKey]
	caCert := secret.Data["ca.crt"]

	// Get TLS secret of management ingress service, then get CA cert for OCP route
	secret, err = ingressRequest.GetSecret(TLSSecretName)
	if err != nil {
		return err
	}
	destinationCAcert := secret.Data["ca.crt"]

	route := NewRoute(
		RouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ServiceName,
		ingressRequest.managementIngress.Status.Host,
		cert,
		key,
		caCert,
		destinationCAcert,
	)

	// Create route resource
	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, route, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Route: %v", err)
	}

	klog.Infof("Creating route: %s for %q.", route.ObjectMeta.Name, ingressRequest.managementIngress.ObjectMeta.Name)
	err = ingressRequest.Create(route)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing route for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedRoute", "Successfully created route %q", RouteName)

	if err = ingressRequest.createOrUpdateSecret(ClusterSecretName, os.Getenv(PODNAMESPACE), caCert); err != nil {
		return fmt.Errorf("Unable to create or update secret for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}

//GetRouteURL retrieves the route URL from a given route and namespace
func (ingressRequest *IngressRequest) GetRouteURL(name string) (string, error) {

	foundRoute := &route.Route{}

	if err := ingressRequest.Get(name, ingressRequest.managementIngress.ObjectMeta.Namespace, foundRoute); err != nil {
		if !errors.IsNotFound(err) {
			return "", err
		}
	}

	return fmt.Sprintf("%s%s", "https://", foundRoute.Spec.Host), nil
}

// GetRouteAppDomain ... auto detect route application domain of OCP cluster.
func (ingressRequest *IngressRequest) GetRouteAppDomain() (string, error) {
	klog.Infof("Getting route application domain name from ingress controller config.")

	ing := &operatorv1.IngressController{}
	if err := ingressRequest.Get("default", "openshift-ingress-operator", ing); err != nil {
		return "", err
	}

	if ing != nil {
		appDomain := ing.Status.Domain
		if len(appDomain) > 0 {
			return appDomain, nil
		}
	}

	return "", fmt.Errorf("The router Domain from config of Ingress Controller Operator is empty. See more info: %v", ing)
}
