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
	"os"
	"strings"
	"time"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alph1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
)

//NewCertificate stubs an instance of Certificate
func NewCertificate(name, namespace, secret string, hosts, ips []string, issuer *v1alph1.CertIssuer) *certmanager.Certificate {

	labels := GetCommonLabels()

	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		klog.Errorf("Failure getting watch namespace: %v", err)
		os.Exit(1)
	}

	issuerKind := "Issuer"
	if len(watchNamespace) == 0 {
		issuerKind = "ClusterIssuer"
	}

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
				Kind: issuerKind,
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

	if err := ingressRequest.CreateCert(routeCert); err != nil {
		return err
	}

	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedCertificate", "Successfully created certificate %q", CertName)

	return nil
}

func (ingressRequest *IngressRequest) CreateCert(cert *certmanager.Certificate) error {
	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, cert, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Certificate: %v", err)
	}

	klog.Infof("Creating Certificate: %s for %q.", cert.ObjectMeta.Name, ingressRequest.managementIngress.ObjectMeta.Name)
	err := ingressRequest.Create(cert)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing certificate for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
	}

	return nil
}
