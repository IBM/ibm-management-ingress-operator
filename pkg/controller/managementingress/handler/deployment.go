package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
	operatorv1 "github.com/openshift/api/operator/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

const (
	httpsPort = int32(8443)
	httpPort  = int32(8080)
)

//NewDeployment stubs an instance of a deployment
func NewDeployment(name string, namespace string, podSpec core.PodSpec) *apps.Deployment {
	labels := map[string]string{
		"component": AppName,
		"app":       AppName,
	}
	return &apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: apps.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: labels,
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
						"productID":      "cp-0000001",
						"productName":    "IBM Cloud Platform Common Services",
						"productVersion": "3.3.0",
					},
				},
				Spec: podSpec,
			},
		},
	}
}

func newPodSpec(img, clusterDomain string, resources *core.ResourceRequirements, nodeSelector map[string]string, tolerations []core.Toleration, allowedHostHeader string, fipsEnabled bool) core.PodSpec {
	if resources == nil {
		resources = &core.ResourceRequirements{
			Limits: core.ResourceList{core.ResourceMemory: defaultMemory},
			Requests: core.ResourceList{
				core.ResourceMemory: defaultMemory,
				core.ResourceCPU:    defaultCpuRequest,
			},
		}
	}

	container := core.Container{
		Name:            AppName,
		Image:           img,
		ImagePullPolicy: core.PullIfNotPresent,
		Resources:       *resources,
	}

	container.Ports = []core.ContainerPort{
		core.ContainerPort{
			Name:          "https",
			ContainerPort: httpsPort,
			Protocol:      core.ProtocolTCP,
		},
		core.ContainerPort{
			Name:          "http",
			ContainerPort: httpPort,
			Protocol:      core.ProtocolTCP,
		},
	}

	container.Command = []string{
		"/icp-management-ingress",
		"--default-ssl-certificate=$(POD_NAMESPACE)/icp-management-ingress-tls-secret",
		"--configmap=$(POD_NAMESPACE)/management-ingress",
		"--http-port=8080",
		"--https-port=8443",
	}

	container.Env = []core.EnvVar{
		{Name: "ENABLE_IMPERSONATION", Value: "false"},
		{Name: "APISERVER_SECURE_PORT", Value: "6443"},
		{Name: "CLUSTER_DOMAIN", Value: clusterDomain},
		{Name: "HOST_HEADERS_CHECK_ENABLED", Value: strconv.FormatBool(len(allowedHostHeader) > 0)},
		{Name: "ALLOWED_HOST_HEADERS", Value: allowedHostHeader},
		{Name: "OIDC_ISSUER_URL", ValueFrom: &core.EnvVarSource{
			ConfigMapKeyRef: &core.ConfigMapKeySelector{
				Key: "OIDC_ISSUER_URL",
				LocalObjectReference: core.LocalObjectReference{
					Name: PlatformAuthConfigmap}}}},
		{Name: "WLP_CLIENT_ID", ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				Key: "WLP_CLIENT_ID",
				LocalObjectReference: core.LocalObjectReference{
					Name: PlatformAuthSecret}}}},
		{Name: "POD_NAME", ValueFrom: &core.EnvVarSource{FieldRef: &core.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"}}},
		{Name: "POD_NAMESPACE", ValueFrom: &core.EnvVarSource{FieldRef: &core.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"}}},
		{Name: "FIPS_ENABLED", Value: strconv.FormatBool(fipsEnabled)},
	}

	container.SecurityContext = &core.SecurityContext{
		Privileged:               utils.GetBool(false),
		AllowPrivilegeEscalation: utils.GetBool(false),
	}

	container.LivenessProbe = &core.Probe{
		TimeoutSeconds:      1,
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		FailureThreshold:    10,
		Handler: core.Handler{
			HTTPGet: &core.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Scheme: core.URISchemeHTTP,
			},
		},
	}

	container.ReadinessProbe = &core.Probe{
		TimeoutSeconds:      1,
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		Handler: core.Handler{
			HTTPGet: &core.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Scheme: core.URISchemeHTTP,
			},
		},
	}

	container.VolumeMounts = []core.VolumeMount{
		{Name: "tls-secret", MountPath: "/var/run/secrets/tls"},
	}

	tolerations = utils.AppendTolerations(
		tolerations,
		[]core.Toleration{
			core.Toleration{
				Key:      "node.kubernetes.io/memory-pressure",
				Operator: core.TolerationOpExists,
				Effect:   core.TaintEffectNoSchedule,
			},
			core.Toleration{
				Key:      "node.kubernetes.io/disk-pressure",
				Operator: core.TolerationOpExists,
				Effect:   core.TaintEffectNoSchedule,
			},
		},
	)

	podSpec := core.PodSpec{
		Containers:         []core.Container{container},
		ServiceAccountName: ServiceAccountName,
		NodeSelector:       nodeSelector,
		Tolerations:        tolerations,
	}

	defaultMode := int32(0644)
	podSpec.Volumes = []core.Volume{
		{
			Name: "tls-secret",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName:  TLSSecretName,
					DefaultMode: &defaultMode,
				},
			},
		},
	}

	// podSpec.PriorityClassName = csPriorityClassName
	podSpec.TerminationGracePeriodSeconds = utils.GetInt64(30)

	return podSpec
}

func getClusterDomain(ingressRequest *IngressRequest) (string, error) {
	klog.Infof("Getting cluster domain from DNS config.")

	dns := &operatorv1.DNS{}
	if err := ingressRequest.Get("default", "", dns); err != nil {
		return "", err
	}

	if dns != nil {
		clusterDomain := dns.Status.ClusterDomain
		if len(clusterDomain) > 0 {
			return clusterDomain, nil
		}
	}

	return "", fmt.Errorf("The Cluster Domain from DNS operator config is empty. Check DNS: %v", dns)
}

func (ingressRequest *IngressRequest) CreateOrUpdateDeployment() error {

	klog.Infof("Creating or Updating Deployment: %s for %q.", AppName, ingressRequest.managementIngress.Name)
	imageRepo := strings.Join([]string{
		ingressRequest.managementIngress.Spec.ImageRegistry,
		ingressRequest.managementIngress.Spec.Image.Repository,
	}, "/")
	image := strings.Join([]string{
		imageRepo,
		ingressRequest.managementIngress.Spec.Image.Tag,
	}, ":")

	hostHeader := strings.Join([]string{
		ingressRequest.managementIngress.Spec.AllowedHostHeader,
		ingressRequest.managementIngress.Status.Host,
		ServiceName,
		IAMTokenService,
		strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace}, "."),
		strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace, "svc"}, "."),
		strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace}, "."),
		strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace, "svc"}, "."),
	}, " ")

	clusterDomain, err := getClusterDomain(ingressRequest)
	if err != nil {
		return fmt.Errorf("Failure getting cluster domain: %v", err)
	}

	podSpec := newPodSpec(
		image,
		clusterDomain,
		ingressRequest.managementIngress.Spec.Resources,
		ingressRequest.managementIngress.Spec.NodeSelector,
		ingressRequest.managementIngress.Spec.Tolerations,
		hostHeader,
		ingressRequest.managementIngress.Spec.FIPSEnabled,
	)

	ds := NewDeployment(
		AppName,
		ingressRequest.managementIngress.Namespace,
		podSpec)

	utils.AddOwnerRefToObject(ds, utils.AsOwner(ingressRequest.managementIngress))

	err = ingressRequest.Create(ds)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failure creating deployment %q: %v", AppName, err)
			return fmt.Errorf("Failure creating Deployment: %v", err)
		}

		current := &apps.Deployment{}
		if err = ingressRequest.Get(AppName, ingressRequest.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("Failure getting %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		desired, different := utils.IsDeploymentDifferent(current, ds)
		if !different {
			return nil
		}

		klog.Infof("There is change from Deployment %s. Try to update it.", podSpec)
		err = ingressRequest.Update(desired)
		if err != nil {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failure updating deployment %q: %v", AppName, err)
			return fmt.Errorf("Failure updating %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}
		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "UpdatedDeployment", "Successfully updated deployment %q", AppName)
	} else {
		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedDeployment", "Successfully created deployment %q", AppName)
	}

	return nil
}

//GetDeploymentList lists DS in namespace with given selector
func (ingressRequest *IngressRequest) GetDeploymentList(selector map[string]string) (*apps.DeploymentList, error) {
	list := &apps.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: apps.SchemeGroupVersion.String(),
		},
	}

	err := ingressRequest.List(
		selector,
		list,
	)

	return list, err
}

func (ingressRequest *IngressRequest) GetDeploymentPods(selector map[string]string) (*core.PodList, error) {
	list := &core.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: apps.SchemeGroupVersion.String(),
		},
	}

	err := ingressRequest.List(
		selector,
		list,
	)

	return list, err
}

//RemoveDaemonset with given name and namespace
func (ingressRequest *IngressRequest) RemoveDaemonset(name string) error {

	deployment := &apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: apps.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.Namespace,
		},
		Spec: apps.DeploymentSpec{},
	}

	klog.Infof("Deleting Deployment for %q.", ingressRequest.managementIngress.Name)
	err := ingressRequest.Delete(deployment)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %q deployment %v", name, err)
	}

	return nil
}

func (ingressRequest *IngressRequest) waitForDeploymentReady(ds *apps.Deployment) error {

	err := wait.Poll(5*time.Second, 2*time.Second, func() (done bool, err error) {
		err = ingressRequest.Get(ds.Name, ingressRequest.managementIngress.ObjectMeta.Namespace, ds)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, fmt.Errorf("Failed to get Fluentd deployment: %v", err)
			}
			return false, err
		}

		if int(ds.Status.ReadyReplicas) == int(ds.Status.Replicas) {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}
