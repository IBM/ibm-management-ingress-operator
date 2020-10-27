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
	"fmt"
	"strings"
	"time"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
)

//NewCertificate stubs an instance of Certificate
func NewCertificate(name, namespace, secret string, hosts, ips []string, issuer *operatorv1alpha1.CertIssuer) *certmanager.Certificate {

	labels := GetCommonLabels()

	return &certmanager.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "certmanager.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: certmanager.CertificateSpec{
			CommonName:  AppName,
			Duration:    &metav1.Duration{Duration: 8760 * time.Hour},
			RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
			SecretName:  secret,
			IssuerRef: certmanager.ObjectReference{
				Kind: string(issuer.Kind),
				Name: issuer.Name,
			},
			DNSNames:    hosts,
			IPAddresses: ips,
			Usages:      []certmanager.KeyUsage{certmanager.UsageDigitalSignature, certmanager.UsageKeyEncipherment, certmanager.UsageServerAuth},
		},
	}
}

func getDefaultDNSNames(service, namespace string) []string {
	dns1 := service
	dns2 := strings.Join([]string{dns1, namespace}, ".")
	dns3 := strings.Join([]string{dns2, "svc"}, ".")

	return []string{dns1, dns2, dns3}
}

func (ingressRequest *IngressRequest) CreateOrUpdateCertificates() error {
	// Create certificate for management ingress
	defaultDNS := getDefaultDNSNames(ServiceName, ingressRequest.managementIngress.ObjectMeta.Namespace)
	DNS := ingressRequest.managementIngress.Spec.Cert.DNSNames

	cert := NewCertificate(
		CertName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		TLSSecretName,
		append(defaultDNS, DNS...),
		ingressRequest.managementIngress.Spec.Cert.IPAddresses,
		&ingressRequest.managementIngress.Spec.Cert.Issuer,
	)

	if err := ingressRequest.CreateCert(cert); err != nil {
		return err
	}

	// Create TLS certificate for management ingress route
	routeCert := NewCertificate(
		RouteCert,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		RouteSecret,
		[]string{ingressRequest.managementIngress.Status.Host},
		[]string{},
		&ingressRequest.managementIngress.Spec.Cert.Issuer,
	)

	err := ingressRequest.CreateCert(routeCert)

	return err
}

func (ingressRequest *IngressRequest) CreateCert(cert *certmanager.Certificate) error {
	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, cert, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Certificate: %v", err)
	}

	err := ingressRequest.Create(cert)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}

		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "CreatedCertificate", "Failed to create certificate %q", cert.ObjectMeta.Name)
		return fmt.Errorf("failure creating certificate: %v", err)
	}

	klog.Infof("Created Certificate: %s.", cert.ObjectMeta.Name)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedCertificate", "Successfully created certificate %q", cert.ObjectMeta.Name)

	return nil
}
