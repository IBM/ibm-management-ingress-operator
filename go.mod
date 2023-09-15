module github.com/IBM/ibm-management-ingress-operator

go 1.13

require (
	github.com/IBM/controller-filtered-cache v0.3.4
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/prometheus/client_golang v1.14.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.11.1
)

require (
	github.com/ibm/ibm-cert-manager-operator v0.0.0-20220602233809-3a62073266c7
	golang.org/x/net v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20220521103104-8f96da9f5d5e // indirect
)

replace (
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	k8s.io/client-go => k8s.io/client-go v0.23.5
)

replace k8s.io/api => k8s.io/api v0.23.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.23.11-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.23.5

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.5

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.5

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.5

replace k8s.io/code-generator => k8s.io/code-generator v0.23.6-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.23.5

replace k8s.io/component-helpers => k8s.io/component-helpers v0.23.5

replace k8s.io/controller-manager => k8s.io/controller-manager v0.23.5

replace k8s.io/cri-api => k8s.io/cri-api v0.23.14-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.5

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.5

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.5

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.5

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.5

replace k8s.io/kubectl => k8s.io/kubectl v0.23.5

replace k8s.io/kubelet => k8s.io/kubelet v0.23.5

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.5

replace k8s.io/metrics => k8s.io/metrics v0.23.5

replace k8s.io/mount-utils => k8s.io/mount-utils v0.23.14-rc.0

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.5

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.5

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.5

replace k8s.io/sample-controller => k8s.io/sample-controller v0.23.5
