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
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagementIngressSpec defines the desired state of ManagementIngress
type ManagementIngressSpec struct {
	ManagementState   ManagementState              `json:"managementState"`
	ImageRegistry     string                       `json:"imageRegistry"`
	Image             OperandImage                 `json:"image,omitempty"`
	Replicas          int32                        `json:"replicas,omitempty"`
	Resources         *corev1.ResourceRequirements `json:"resources,omitempty"`
	NodeSelector      map[string]string            `json:"nodeSelector,omitempty"`
	Tolerations       []corev1.Toleration          `json:"tolerations,omitempty"`
	AllowedHostHeader string                       `json:"allowedHostHeader,omitempty"`
	Cert              *Cert                        `json:"cert"`
	RouteHost         string                       `json:"routeHost"`
	Config            map[string]string            `json:"config,omitempty"`
	FIPSEnabled       bool                         `json:"fipsEnabled,omitempty"`
	IgnoreRouteCert   bool                         `json:"ignoreRouteCert,omitempty"`
}

type OperandImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type Cert struct {
	// +kubebuilder:validation:Optional
	Issuer CertIssuer `json:"issuer"`
	// +kubebuilder:validation:Optional
	NamespacedIssuer CertIssuer `json:"namespacedIssuer"`
	CommonName       string     `json:"repository,omitempty"`
	DNSNames         []string   `json:"dnsNames,omitempty"`
	IPAddresses      []string   `json:"ipAddresses,omitempty"`
}

type CertIssuer struct {
	Name string     `json:"name"`
	Kind IssuerKind `json:"kind"`
}

type IssuerKind string

const (
	ClusterIssuer IssuerKind = "ClusterIssuer"
	Issuer        IssuerKind = "Issuer"
)

type ConditionList []Condition

// ManagementIngressStatus defines the observed state of ManagementIngress
type ManagementIngressStatus struct {
	Conditions map[string]ConditionList `json:"condition,omitempty"`
	PodState   PodStateMap              `json:"podstate"`
	Host       string                   `json:"host"`
	State      OperandState             `json:"operandState"`
}

type OperandState struct {
	Status  StatusType `json:"status"`
	Message string     `json:"message"`
}

type StatusType string

const (
	StatusFailed     StatusType = "Failed"
	StatusSuccessful StatusType = "Successful"
	StatusDeploying  StatusType = "Deploying"
)

type ManagementState string

const (
	// Managed means that the operator is actively managing its resources and trying to keep the component active.
	// It will only upgrade the component if it is safe to do so
	ManagementStateManaged ManagementState = "Managed"
	// Unmanaged means that the operator will not take any action related to the component
	ManagementStateUnmanaged ManagementState = "Unmanaged"
)

type PodStateMap map[PodStateType][]string

type Condition struct {
	Type               ConditionType   `json:"type"`
	Status             ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time     `json:"lastTransitionTime"`
	Reason             string          `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	Message            string          `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type ConditionType string

const (
	ResourceCreating         ConditionType = "ResourceCreating"
	WaitingResource          ConditionType = "WaitingResource"
	ResourceFailedOnCreation ConditionType = "ResourceFailedOnCreation"
	DiscoveringClusterInfo   ConditionType = "DiscoveringClusterInfo"
)

type PodStateType string

const (
	PodStateTypeReady    PodStateType = "ready"
	PodStateTypeNotReady PodStateType = "notReady"
	PodStateTypeFailed   PodStateType = "failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ManagementIngress is the Schema for the managementingresses API
type ManagementIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagementIngressSpec   `json:"spec,omitempty"`
	Status ManagementIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagementIngressList contains a list of ManagementIngress
type ManagementIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagementIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagementIngress{}, &ManagementIngressList{})
}
