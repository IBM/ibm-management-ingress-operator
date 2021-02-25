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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	route "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//NewRoute stubs an instance of a Route
func NewRoute(name, namespace, serviceName, routeHost string, cert, key, caCert, destinationCAcert []byte) *route.Route {

	labels := GetCommonLabels()
	weight := int32(100)

	r := &route.Route{
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
			Port: &route.RoutePort{
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "https",
				},
			},
			To: route.RouteTargetReference{
				Name:   serviceName,
				Kind:   "Service",
				Weight: &weight,
			},
			WildcardPolicy: route.WildcardPolicyNone,
		},
	}

	if len(cert) > 0 && len(key) > 0 && len(caCert) > 0 && len(destinationCAcert) > 0 {
		// SSL termination is reencrypt
		r.Spec.TLS = &route.TLSConfig{
			Termination:                   route.TLSTerminationReencrypt,
			InsecureEdgeTerminationPolicy: route.InsecureEdgeTerminationPolicyRedirect,
			Certificate:                   string(cert),
			Key:                           string(key),
			CACertificate:                 string(caCert),
			DestinationCACertificate:      string(destinationCAcert),
		}
	} else {
		// SSL termination is passthrough
		r.Spec.TLS = &route.TLSConfig{
			Termination:                   route.TLSTerminationPassthrough,
			InsecureEdgeTerminationPolicy: route.InsecureEdgeTerminationPolicyRedirect,
		}
	}

	return r
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

func waitForRouteSecret(r *IngressRequest, name string, stopCh <-chan struct{}) (*core.Secret, error) {
	klog.Infof("Waiting for secret: %s ...", name)
	s := &core.Secret{}

	err := wait.PollImmediateUntil(2*time.Second, func() (done bool, err error) {
		if err := r.Get(name, r.managementIngress.ObjectMeta.Namespace, s); err != nil {
			return false, nil
		}

		return true, nil
	}, stopCh)

	return s, err
}

// create ibmcloud-cluster-ca-cert
func createClusterCACert(i *IngressRequest, secretName, ns string, caCert []byte) error {
	// create ibmcloud-cluster-ca-cert
	clusterSecret := NewSecret(secretName, ns, caCert)

	if err := controllerutil.SetControllerReference(i.managementIngress, clusterSecret, i.scheme); err != nil {
		klog.Errorf("Error setting controller reference on secret: %v", err)
	}
	err := i.Create(clusterSecret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failure creating secret for %q: %v", i.managementIngress.ObjectMeta.Name, err)
		}

		klog.Infof("Trying to update secret: %s as it already existed.", secretName)
		// Update config
		current := &core.Secret{}
		err := i.Get(secretName, ns, current)
		if err != nil {
			return fmt.Errorf("failure getting secret: %q  for %q: %v", secretName, i.managementIngress.Name, err)
		}

		// no data change, just return
		if reflect.DeepEqual(clusterSecret.Data, current.Data) {
			klog.Infof("No change found from the secret: %s, skip updating current secret.", secretName)
			return nil
		}

		json, _ := json.Marshal(clusterSecret)
		klog.Infof("Found change from secret %s, trying to update it.", json)
		current.Data = clusterSecret.Data

		// Apply the latest change to configmap
		if err = i.Update(current); err != nil {
			return fmt.Errorf("failure updating secret: %v for %q: %v", secretName, i.managementIngress.Name, err)
		}
	}

	klog.Infof("Created secret: %s.", secretName)
	i.recorder.Eventf(i.managementIngress, "Normal", "CreatedSecret", "Successfully created or updated secret %q", secretName)

	return nil
}

func handleUpdate(i *IngressRequest, current *route.Route, src *route.Route) error {
	isChange := false
	name := src.ObjectMeta.Name

	// Check owner reference
	err := controllerutil.SetControllerReference(i.managementIngress, current, i.scheme)
	if err != nil {
		if _, ok := err.(*controllerutil.AlreadyOwnedError); !ok {
			klog.Errorf("Error updating owner reference for route: %v", err)
			return err
		}
	} else {
		// Successfully set owner reference.
		isChange = true
	}

	if equal := reflect.DeepEqual(current.Spec, src.Spec); !equal {
		klog.Infof("Found change for route: %s, trying to update it ...", name)
		current.Spec = src.Spec
		isChange = true
	}

	if isChange {
		err := i.Update(current)
		if err != nil {
			i.recorder.Eventf(i.managementIngress, "Warning", "UpdateRoute", "Failed to update route %q", name)
			return fmt.Errorf("failure updating route: %v", err)
		}

		klog.Infof("Updated route: %s.", name)
		i.recorder.Eventf(i.managementIngress, "Normal", "UpdateRoute", "Successfully updated route %q", name)
	}

	return nil
}

func handleCreate(i *IngressRequest, r *route.Route) error {
	name := r.ObjectMeta.Name

	if err := controllerutil.SetControllerReference(i.managementIngress, r, i.scheme); err != nil {
		klog.Errorf("Error setting owner reference for route: %v", err)
		return err
	}

	// Try to create the route resource
	err := i.Create(r)
	if err != nil {
		i.recorder.Eventf(i.managementIngress, "Warning", "CreateRoute", "Failed to create route %q", name)
		return fmt.Errorf("failure creating %s route: %v", name, err)
	}

	klog.Infof("Created route: %s.", name)
	i.recorder.Eventf(i.managementIngress, "Normal", "CreateRoute", "Successfully created route %q", name)
	return nil
}

func syncRoute(i *IngressRequest, r *route.Route) error {

	name := r.ObjectMeta.Name
	ns := r.ObjectMeta.Namespace
	current := &route.Route{}

	err := i.Get(name, ns, current)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Error getting route: %v", err)
		return fmt.Errorf("failure getting current route: %v", err)
	}

	// Create the route if it's not found.
	if err != nil && errors.IsNotFound(err) {
		err := handleCreate(i, r)
		return err
	}

	// Try to update the route
	err = handleUpdate(i, current, r)
	return err
}

func getRouteCertificate(i *IngressRequest, ns string) ([]byte, []byte, []byte, []byte, error) {
	var cert, key, caCert, destinationCAcert []byte

	// Wait for route secret before creating route. Just avoid the case reconciling failed many times.
	stop := WaitForTimeout(10 * time.Minute)
	secret, err := waitForRouteSecret(i, RouteSecret, stop)
	if err != nil {
		return cert, key, caCert, destinationCAcert, err
	}

	cert = secret.Data[core.TLSCertKey]
	key = secret.Data[core.TLSPrivateKeyKey]
	caCert = secret.Data["ca.crt"]

	// Get TLS secret of management ingress service, then get CA cert for OCP route
	ingressSecret := &core.Secret{}
	if err := i.Get(TLSSecretName, ns, ingressSecret); err != nil {
		return cert, key, caCert, destinationCAcert, err
	}
	destinationCAcert = ingressSecret.Data["ca.crt"]

	return cert, key, caCert, destinationCAcert, nil
}

func (ingressRequest *IngressRequest) CreateOrUpdateRoute() error {

	// Get data from route certificate
	cert, key, caCert, destinationCAcert, err := getRouteCertificate(ingressRequest, ingressRequest.managementIngress.ObjectMeta.Namespace)
	if err != nil {
		return err
	}

	// Create or update secret ibmcloud-cluster-ca-cert
	if err := createClusterCACert(ingressRequest, ClusterSecretName, os.Getenv(PODNAMESPACE), caCert); err != nil {
		return fmt.Errorf("failure creating or updating secret: %v", err)
	}

	// Create cp-console route
	consoleRoute := NewRoute(
		ConsoleRouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ServiceName,
		ingressRequest.managementIngress.Status.Host,
		cert,
		key,
		caCert,
		destinationCAcert,
	)

	if err := syncRoute(ingressRequest, consoleRoute); err != nil {
		return err
	}

	// create cp-proxy route
	proxyRouteHost, err := ingressRequest.GetProxyRouteHost()
	if err != nil {
		return fmt.Errorf("failure getting proxy route host: %v", err)
	}
	proxyRoute := NewRoute(
		ProxyRouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ProxyServiceName,
		proxyRouteHost,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	)

	if err := syncRoute(ingressRequest, proxyRoute); err != nil {
		return err
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
	ing := &operatorv1.IngressController{}
	clusterClient, err := createOrGetClusterClient()
	if err != nil {
		return "", fmt.Errorf("failure creating or getting cluster client: %v", err)
	}
	if err := clusterClient.Get(context.TODO(), types.NamespacedName{Name: "default", Namespace: "openshift-ingress-operator"}, ing); err != nil {
		return "", err
	}

	if ing != nil {
		appDomain := ing.Status.Domain
		if len(appDomain) > 0 {
			return appDomain, nil
		}
	}

	return "", fmt.Errorf("the router Domain from config of Ingress Controller Operator is empty. See more info: %v", ing)
}

// Get the host for the cp-proxy route
func (ingressRequest *IngressRequest) GetProxyRouteHost() (string, error) {

	if specifiedProxyRouteHost := ingressRequest.managementIngress.Spec.ProxyRouteHost; len(specifiedProxyRouteHost) > 0 {
		klog.Infof("Got proxyRouteHost %s from CR", specifiedProxyRouteHost)
		return specifiedProxyRouteHost, nil
	}

	// User did not specify proxy route host. Get the default one.
	appDomain, err := ingressRequest.GetRouteAppDomain()
	if err != nil {
		return "", err
	}

	return strings.Join([]string{ProxyRouteName, appDomain}, "."), nil
}
