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
	"reflect"
	"strings"
	"time"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
	klog.Info("starting CreateOrUpdateCertificates...")
	// Create certificate for management ingress
	defaultDNS := getDefaultDNSNames(ServiceName, ingressRequest.managementIngress.ObjectMeta.Namespace)
	DNS := ingressRequest.managementIngress.Spec.Cert.DNSNames

	issuer := ingressRequest.managementIngress.Spec.Cert.NamespacedIssuer
	if issuer == (operatorv1alpha1.CertIssuer{}) {
		klog.Info("empty issuer in managementIngress CR, using default issuer name and kind to create the certificate...")
		issuer = operatorv1alpha1.CertIssuer{
			Name: DefaultCAIssuerName,
			Kind: operatorv1alpha1.IssuerKind(DefaultCAIssuerKind),
		}
	}

	cert := NewCertificate(
		CertName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		TLSSecretName,
		append(defaultDNS, DNS...),
		ingressRequest.managementIngress.Spec.Cert.IPAddresses,
		&issuer,
	)

	if err := ingressRequest.CreateOrUpdateCert(cert); err != nil {
		return err
	}

	// Create TLS certificate for management ingress route
	routeCert := NewCertificate(
		RouteCert,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		RouteSecret,
		[]string{ingressRequest.managementIngress.Status.Host},
		[]string{},
		&issuer,
	)

	return ingressRequest.CreateOrUpdateCert(routeCert)
}

func (ingressRequest *IngressRequest) CreateOrUpdateCert(cert *certmanager.Certificate) error {
	klog.Info("starting CreateOrUpdateCert poll...")
	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, cert, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Certificate: %v", err)
	}

	stop := WaitForTimeout(10 * time.Minute)
	if err := waitForCert(ingressRequest, cert, stop); err != nil {
		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "CreatedOrUpdateCertificate", "Failed to create or update certificate %q", cert.ObjectMeta.Name)
		return fmt.Errorf("failure creating certificate: %s %v", cert.ObjectMeta.Name, err)
	}

	klog.Infof("Created or update certificate: %s.", cert.ObjectMeta.Name)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedCertificate", "Successfully created certificate %q", cert.ObjectMeta.Name)

	return nil
}

func waitForCert(r *IngressRequest, cert *certmanager.Certificate, stopCh <-chan struct{}) error {

	err := wait.PollImmediateUntil(2*time.Second, func() (done bool, err error) {
		klog.V(4).Infof("Try to create certificate: %v", cert)
		if err := r.Create(cert); err != nil {
			if errors.IsAlreadyExists(err) {
				klog.Infof("Trying to update certificate: %s as it already existed.", cert.ObjectMeta.Name)
				current := &certmanager.Certificate{}
				if err = r.Get(cert.ObjectMeta.Name, r.managementIngress.ObjectMeta.Namespace, current); err != nil {
					return false, fmt.Errorf("failure getting certificate: %q for %q: %v", cert.ObjectMeta.Name, r.managementIngress.Name, err)
				}
				if equal := reflect.DeepEqual(current.Spec, cert.Spec); equal {
					klog.Infof("No change found from certificate: %s.", cert.ObjectMeta.Name)
					return true, nil
				}
				klog.Infof("Found change for certificate: %s. Trying to update it.", cert.ObjectMeta.Name)
				current.Spec = cert.Spec
				err = r.Update(current)
				if err != nil {
					r.recorder.Eventf(r.managementIngress, "Warning", "UpdatedCertificate", "Failed to update certificate: %s", cert.ObjectMeta.Name)
					return false, fmt.Errorf("failure updating certificate: %q for %q: %v", cert.ObjectMeta.Name, r.managementIngress.Name, err)
				}
				r.recorder.Eventf(r.managementIngress, "Normal", "UpdatedCertificate", "Successfully updated certificate %q", cert.ObjectMeta.Name)
				return true, nil
			}

			klog.V(4).Infof("Failed to create or update certificate: %+v, retrying again ...", err)
			return false, nil
		}

		return true, nil
	}, stopCh)

	return err
}
