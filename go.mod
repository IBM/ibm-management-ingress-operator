module github.com/IBM/ibm-management-ingress-operator

go 1.13

require (
	github.com/IBM/controller-filtered-cache v0.2.0
	github.com/jetstack/cert-manager v0.10.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	go.uber.org/zap v1.13.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.6.2
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	golang.org/x/crypto v0.7.0 // indirect
)

replace (
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	k8s.io/client-go => k8s.io/client-go v0.18.8
)
