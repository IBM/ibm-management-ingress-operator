package handler

import (
	"fmt"

	route "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/IBM/management-ingress-operator/pkg/utils"
)

//NewRoute stubs an instance of a Route
func NewRoute(name, namespace, serviceName, routeHost string, cert, key, caCert, destinationCAcert []byte) *route.Route {
	return &route.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: route.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"component": AppName,
			},
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

func (ingressRequest *IngressRequest) CreateOrUpdateRoute() error {
	klog.Infof("Creating route for %q.", ingressRequest.managementIngress.ObjectMeta.Name)

	// Get TLS secret for OCP route
	err, secret := ingressRequest.GetSecret(RouteSecret)
	if err != nil {
		return err
	}
	cert := secret.Data[core.TLSCertKey]
	key := secret.Data[core.TLSPrivateKeyKey]
	caCert := secret.Data["ca.crt"]

	// Get TLS secret of management ingress service, then get CA cert for OCP route
	err, secret = ingressRequest.GetSecret(TLSSecretName)
	if err != nil {
		return err
	}
	destinationCAcert := secret.Data["ca.crt"]

	route := NewRoute(
		RouteName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ServiceName,
		ingressRequest.managementIngress.Spec.RouteHost,
		cert,
		key,
		caCert,
		destinationCAcert,
	)

	// Create route resource
	utils.AddOwnerRefToObject(route, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Trying to create route %q for %q.", route.ObjectMeta.Name, ingressRequest.managementIngress.ObjectMeta.Name)
	err = ingressRequest.Create(route)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing route for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
	}
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedRoute", "Successfully created route %q", RouteName)

	return nil
}

//GetRouteURL retrieves the route URL from a given route and namespace
func (ingressRequest *IngressRequest) GetRouteURL(name string) (string, error) {

	foundRoute := &route.Route{}

	if err := ingressRequest.Get(name, foundRoute); err != nil {
		if !errors.IsNotFound(err) {
			return "", err
		}
	}

	return fmt.Sprintf("%s%s", "https://", foundRoute.Spec.Host), nil
}

//RemoveRoute with given name and namespace
func (ingressRequest *IngressRequest) RemoveRoute(name string) error {

	route := &route.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: route.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
		},
		Spec: route.RouteSpec{},
	}

	klog.Infof("Deleting route for %q.", ingressRequest.managementIngress.ObjectMeta.Name)
	err := ingressRequest.Delete(route)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %q route %v", name, err)
	}

	return nil
}
