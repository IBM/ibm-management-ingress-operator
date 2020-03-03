package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagementIngressSpec defines the desired state of ManagementIngress
type ManagementIngressSpec struct {
	ManagementState   ManagementState          `json:"managementState"`
	Image             OperandImage             `json:"image"`
	Resources         *v1.ResourceRequirements `json:"resources,omitempty"`
	NodeSelector      map[string]string        `json:"nodeSelector,omitempty"`
	Tolerations       []v1.Toleration          `json:"tolerations,omitempty"`
	AllowedHostHeader string                   `json:"allowedHostHeader,omitempty"`
	Cert              *Cert                    `json:"cert"`
	RouteHost         string                   `json:"routeHost"`
	Config            map[string]string        `json:"config,omitempty"`
	FIPSEnabled       bool                     `json:"fipsEnabled,omitempty"`
}

type OperandImage struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type Cert struct {
	Issuer      CertIssuer `json:"issuer"`
	CommonName  string     `json:"repository,omitempty"`
	DNSNames    []string   `json:"dnsNames,omitempty"`
	IPAddresses []string   `json:"ipAddresses,omitempty"`
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
	PodState   PodStateMap            `json:"podstate"`
}

// ManagementIngress is the Schema for the managementingresses API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managementingresses,scope=Namespaced
type ManagementIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagementIngressSpec   `json:"spec,omitempty"`
	Status ManagementIngressStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagementIngressList contains a list of ManagementIngress
type ManagementIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagementIngress `json:"items"`
}

type ManagementState string

const (
	// Managed means that the operator is actively managing its resources and trying to keep the component active.
	// It will only upgrade the component if it is safe to do so
	ManagementStateManaged ManagementState = "Managed"
	// Unmanaged means that the operator will not take any action related to the component
	ManagementStateUnmanaged ManagementState = "Unmanaged"
)

type PodStateMap map[PodStateType][]string

type PodStateType string

type Condition struct {
	Type               ConditionType      `json:"type"`
	Status             v1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime"`
	Reason             string             `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	Message            string             `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// ClusterConditionType is a valid value for ClusterCondition.Type
type ConditionType string

const (
	ContainerWaiting    ConditionType = "ContainerWaiting"
	ContainerTerminated ConditionType = "ContainerTerminated"
	Unschedulable       ConditionType = "Unschedulable"
)

const (
	PodStateTypeReady    PodStateType = "ready"
	PodStateTypeNotReady PodStateType = "notReady"
	PodStateTypeFailed   PodStateType = "failed"
)

func init() {
	SchemeBuilder.Register(&ManagementIngress{}, &ManagementIngressList{})
}
