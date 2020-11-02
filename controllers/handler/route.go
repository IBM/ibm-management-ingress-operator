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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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
				Name: serviceName,
				Kind: "Service",
			},
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
func (ingressRequest *IngressRequest) createClusterCACert(secretName, namespace string, caCert []byte) error {
	// create ibmcloud-cluster-ca-cert
	clusterSecret := NewSecret(secretName, namespace, caCert)

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, clusterSecret, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Secret: %v", err)
	}
	err := ingressRequest.Create(clusterSecret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failure creating secret for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
		}

		klog.Infof("Trying to update Secret: %s as it already existed.", secretName)
		// Update config
		current := &core.Secret{}
		err := ingressRequest.Get(secretName, namespace, current)
		if err != nil {
			return fmt.Errorf("failure getting Secret: %q  for %q: %v", secretName, ingressRequest.managementIngress.Name, err)
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
			return fmt.Errorf("failure updating Secret: %v for %q: %v", secretName, ingressRequest.managementIngress.Name, err)
		}
	}

	klog.Infof("Created secret: %s.", secretName)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedSecret", "Successfully created or updated secret %q", secretName)

	return nil
}

func (ingressRequest *IngressRequest) CreateOrUpdateRoute() error {
	// Wait for route secret before creating route. Just avoid the case reconciling failed many times.
	stop := WaitForTimeout(10 * time.Minute)
	secret, err := waitForRouteSecret(ingressRequest, RouteSecret, stop)
	if err != nil {
		return err
	}
	cert := secret.Data[core.TLSCertKey]
	key := secret.Data[core.TLSPrivateKeyKey]
	caCert := secret.Data["ca.crt"]

	// Get TLS secret of management ingress service, then get CA cert for OCP route
	ingressSecret := &core.Secret{}
	if err := ingressRequest.Get(TLSSecretName, ingressRequest.managementIngress.ObjectMeta.Namespace, ingressSecret); err != nil {
		return err
	}
	destinationCAcert := ingressSecret.Data["ca.crt"]

	// Create or update secret ibmcloud-cluster-ca-cert
	if err := ingressRequest.createClusterCACert(ClusterSecretName, os.Getenv(PODNAMESPACE), caCert); err != nil {
		return fmt.Errorf("failure creating or updating secret: %v", err)
	}

	// Create cp-console route
	consoleRoute := NewRoute(
		RouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ServiceName,
		ingressRequest.managementIngress.Status.Host,
		cert,
		key,
		caCert,
		destinationCAcert,
	)

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, consoleRoute, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting owner reference on cp-console Route: %v", err)
	}

	if err := ingressRequest.Create(consoleRoute); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update route if it exits.
			// Handle the case that route-tls-secret was renewed or updated.
			if err := ingressRequest.Get(RouteName, ingressRequest.managementIngress.ObjectMeta.Namespace, consoleRoute); err != nil {
				klog.Errorf("Error getting route cp-console: %v", err)
				return fmt.Errorf("failure updating cp-console route: %v", err)
			}

			// Update route certificate
			consoleRoute.Spec.TLS.CACertificate = string(caCert)
			consoleRoute.Spec.TLS.Certificate = string(cert)
			consoleRoute.Spec.TLS.Key = string(key)

			klog.Infof("cp-console route already exists. Trying to update with latest route certificate.")
			if err := ingressRequest.Update(consoleRoute); err != nil {
				ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdateRoute", "Failed to update route %q", RouteName)
				return fmt.Errorf("failure updating cp-console route: %v", err)
			}
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "UpdateRoute", "Successfully updated route %q", RouteName)
		} else {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "CreateRoute", "Failed to create route %q", RouteName)
			return fmt.Errorf("failure creating cp-console route: %v", err)
		}
	} else {
		klog.Infof("Created route: %s.", RouteName)
		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreateRoute", "Successfully created route %q", RouteName)
	}

	// Create cp-proxy route
	baseDomain, err := ingressRequest.GetRouteAppDomain()
	if err != nil {
		return fmt.Errorf("failure getting route base domain: %v", err)
	}

	proxyRoute := NewRoute(
		ProxyRouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ProxyServiceName,
		ProxyRouteName+"."+baseDomain,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	)

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, proxyRoute, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on cp-proxy Route: %v", err)
	}

	err = ingressRequest.Create(proxyRoute)
	if err != nil {
		// Update the route with owner reference, then when management ingress CR was removed the route resource will be GCed by k8s.
		if errors.IsAlreadyExists(err) {
			if err := ingressRequest.Get(ProxyRouteName, ingressRequest.managementIngress.ObjectMeta.Namespace, proxyRoute); err != nil {
				klog.Errorf("Error getting route cp-proxy: %v", err)
				return nil
			}

			existingRefs := proxyRoute.GetOwnerReferences()
			if len(existingRefs) == 0 {
				klog.Infof("Route: %s exists. Trying to update it with owner reference", ProxyRouteName)
				if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, proxyRoute, ingressRequest.scheme); err != nil {
					klog.Errorf("Error setting controller reference on cp-proxy Route: %v", err)
				} else if err := ingressRequest.Update(proxyRoute); err != nil {
					ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdateRoute", "Failed to update route: %q owner reference", ProxyRouteName)
					klog.Errorf("Error updating cp-proxy Route for owner reference: %v", err)
				}

				return nil
			}
		} else {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "CreateRoute", "Failed to create route %q", ProxyRouteName)
			return fmt.Errorf("failure creating cp-proxy route: %v", err)
		}
	}

	klog.Infof("Created route: %s.", ProxyRouteName)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreateRoute", "Successfully created route %q", ProxyRouteName)
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
