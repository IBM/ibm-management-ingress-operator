package handler

import (
	"fmt"
	"strings"
	"time"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
)

//NewCertificate stubs an instance of Certificate
func NewCertificate(name string, namespace string, certIssuer string, secret string, service string) *certmanager.Certificate {
	dns1 := service
	dns2 := strings.Join([]string{dns1, namespace}, ".")
	dns3 := strings.Join([]string{dns2, "svc"}, ".")

	return &certmanager.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "certmanager.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"component": AppName,
			},
		},
		Spec: certmanager.CertificateSpec{
			CommonName:  AppName,
			Duration:    &metav1.Duration{Duration: 8760 * time.Hour},
			RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
			SecretName:  secret,
			IssuerRef: certmanager.ObjectReference{
				Kind: "ClusterIssuer",
				Name: certIssuer,
			},
			DNSNames: []string{
				dns1,
				dns2,
				dns3,
			},
		},
	}
}

func (ingressRequest *IngressRequest) CreateOrUpdateCertificates() error {
	// Create certificate for management ingress
	cert := NewCertificate(
		CertName,
		ingressRequest.managementIngress.ObjectMeta.Namespace,
		ingressRequest.managementIngress.Spec.Cert.Issuer,
		TLSSecretName,
		ServiceName,
	)

	if err := ingressRequest.CreateCert(cert); err != nil {
		return err
	}

	// Create TLS certificate for management ingress route
	routeCert := &certmanager.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "certmanager.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      RouteCert,
			Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
			Labels: map[string]string{
				"component": AppName,
			},
		},
		Spec: certmanager.CertificateSpec{
			CommonName:  AppName,
			Duration:    &metav1.Duration{Duration: 8760 * time.Hour},
			RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
			SecretName:  RouteSecret,
			IssuerRef: certmanager.ObjectReference{
				Kind: "ClusterIssuer",
				Name: ingressRequest.managementIngress.Spec.Cert.Issuer,
			},
			DNSNames: []string{
				ingressRequest.managementIngress.Spec.RouteHost,
			},
		},
	}

	if err := ingressRequest.CreateCert(routeCert); err != nil {
		return err
	}

	// Delete the secret to make sure the TLS cert will be freshed by cert manager. Of course all related pods which
	// refer to the secret need to be restarted.
	// secret := &core.Secret{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "Secret",
	// 		APIVersion: core.SchemeGroupVersion.String(),
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      TLSSecretName,
	// 		Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
	// 	},
	// 	Data: map[string][]byte{},
	// }

	// klog.V(4).Infof("Deleting related secret %q for the Certificate to force a refresh.", TLSSecretName)
	// err = ingressRequest.Delete(secret)
	// if err != nil && !errors.IsNotFound(err) {
	// 	klog.Errorf("Failure deleting secret to force a refresh of cert: %v", err)
	// }

	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedCertificate", "Successfully created certificate %q", CertName)

	return nil
}

func (ingressRequest *IngressRequest) CreateCert(cert *certmanager.Certificate) error {
	utils.AddOwnerRefToObject(cert, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Creating Certificate: %s for %q.", CertName, ingressRequest.managementIngress.ObjectMeta.Name)
	err := ingressRequest.Create(cert)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failure constructing certificate for %q: %v", ingressRequest.managementIngress.ObjectMeta.Name, err)
	}

	return nil
}

//RemoveCertificate with a given name and namespace
func (ingressRequest *IngressRequest) RemoveCertificate(name string) error {
	cert := &certmanager.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: "certmanager.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
		},
		Spec: certmanager.CertificateSpec{},
	}

	// Delete certificate
	klog.Infof("Deleting Certificate for %q.", ingressRequest.managementIngress.ObjectMeta.Name)
	err := ingressRequest.Delete(cert)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %v certificate: %v", name, err)
	}

	// Also delete the secret managed by cert
	secret := &core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TLSSecretName,
			Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace,
		},
		Data: map[string][]byte{},
	}

	klog.Infof("Deleting related Secret %s for the certificate too.", TLSSecretName)
	err = ingressRequest.Delete(secret)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Failure deleting secret of cert: %v", err)
	}

	return nil
}
