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
package apis

import (
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	operator "github.com/openshift/api/operator/v1"
	route "github.com/openshift/api/route/v1"
	scc "github.com/openshift/api/security/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
)

var (
	// CertSchemeGroupVersion is group version used to register these objects
	CertSchemeGroupVersion     = schema.GroupVersion{Group: "certmanager.k8s.io", Version: "v1alpha1"}
	RouteSchemeGroupVersion    = schema.GroupVersion{Group: "route.openshift.io", Version: "v1"}
	SCCSchemeGroupVersion      = schema.GroupVersion{Group: "security.openshift.io", Version: "v1"}
	OperatorSchemeGroupVersion = schema.GroupVersion{Group: "operator.openshift.io", Version: "v1"}

	// CertSchemeBuilder is used to add go types to the GroupVersionKind scheme
	CertSchemeBuilder     = &scheme.Builder{GroupVersion: CertSchemeGroupVersion}
	RouteSchemeBuilder    = &scheme.Builder{GroupVersion: RouteSchemeGroupVersion}
	SCCSchemeBuilder      = &scheme.Builder{GroupVersion: SCCSchemeGroupVersion}
	OperatorSchemeBuilder = &scheme.Builder{GroupVersion: OperatorSchemeGroupVersion}
)

func init() {
	CertSchemeBuilder.Register(&certmanager.Certificate{})
	RouteSchemeBuilder.Register(&route.Route{})
	SCCSchemeBuilder.Register(&scc.SecurityContextConstraints{})
	OperatorSchemeBuilder.Register(&operator.IngressController{})
	OperatorSchemeBuilder.Register(&operator.IngressControllerList{})
	OperatorSchemeBuilder.Register(&operator.DNS{})
	OperatorSchemeBuilder.Register(&operator.DNSList{})

	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and bac
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, CertSchemeBuilder.AddToScheme,
		RouteSchemeBuilder.AddToScheme, SCCSchemeBuilder.AddToScheme, OperatorSchemeBuilder.AddToScheme)
}
