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
	"strconv"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
)

const (
	httpsPort = int32(443)
	httpPort  = int32(8080)
)

//NewDeployment stubs an instance of a deployment
func NewDeployment(name string, namespace string, replicas int32, podSpec core.PodSpec) *apps.Deployment {

	labels := GetCommonLabels()
	commAnnotations := GetCommonAnnotations()
	podAnnotations := map[string]string{
		"scheduler.alpha.kubernetes.io/critical-pod": "",
		"clusterhealth.ibm.com/dependencies":         "cert-manager, auth-idp",
	}

	// Merget common annotations with pod specific annotations.
	for k, v := range commAnnotations {
		podAnnotations[k] = v
	}

	return &apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: apps.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: commAnnotations,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
		},
	}
}

func newPodSpec(img, clusterDomain string, resources *core.ResourceRequirements, nodeSelector map[string]string,
	tolerations []core.Toleration, allowedHostHeader string, fipsEnabled bool) core.PodSpec {
	if resources == nil {
		resources = &core.ResourceRequirements{
			Limits: core.ResourceList{
				core.ResourceMemory: defaultMemoryLimit,
				core.ResourceCPU:    defaultCPULimit,
			},
			Requests: core.ResourceList{
				core.ResourceMemory: defaultMemoryRequest,
				core.ResourceCPU:    defaultCPURequest,
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
		{
			Name:          "https",
			ContainerPort: httpsPort,
			Protocol:      core.ProtocolTCP,
		},
		{
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
		"--https-port=443",
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
			{
				Key:      "node.kubernetes.io/memory-pressure",
				Operator: core.TolerationOpExists,
				Effect:   core.TaintEffectNoSchedule,
			},
			{
				Key:      "node.kubernetes.io/disk-pressure",
				Operator: core.TolerationOpExists,
				Effect:   core.TaintEffectNoSchedule,
			},
		},
	)

	matchExpressions := getCommonMatchExpressions()
	podAffinityTerm := core.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: matchExpressions,
		},
		TopologyKey: "kubernetes.io/hostname",
	}

	weightedPodAffinityTerm := core.WeightedPodAffinityTerm{
		PodAffinityTerm: podAffinityTerm,
		Weight:          100,
	}

	podAntiAffinity := &core.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []core.WeightedPodAffinityTerm{weightedPodAffinityTerm},
	}

	affinity := &core.Affinity{PodAntiAffinity: podAntiAffinity}

	podSpec := core.PodSpec{
		Containers:         []core.Container{container},
		ServiceAccountName: ServiceAccountName,
		NodeSelector:       nodeSelector,
		Tolerations:        tolerations,
		Affinity:           affinity,
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

	return "", fmt.Errorf("the Cluster Domain from DNS operator config is empty. Check DNS: %v", dns)
}

func (ingressRequest *IngressRequest) CreateOrUpdateDeployment() error {

	klog.Infof("Creating Deployment: %s for %q.", AppName, ingressRequest.managementIngress.Name)
	image := os.Getenv("OPERAND_IMAGE_DIGEST")
	klog.Infof("Creating Deployment with image: %s.", image)
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
		return fmt.Errorf("failure getting cluster domain: %v", err)
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

	// Set default Management Ingress replica is 1.
	if ingressRequest.managementIngress.Spec.Replicas == 0 {
		ingressRequest.managementIngress.Spec.Replicas = 1
	}

	ds := NewDeployment(
		AppName,
		ingressRequest.managementIngress.Namespace,
		ingressRequest.managementIngress.Spec.Replicas,
		podSpec)

	if err := controllerutil.SetControllerReference(ingressRequest.managementIngress, ds, ingressRequest.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Deployment: %v", err)
	}

	err = ingressRequest.Create(ds)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failure creating deployment %q: %v", AppName, err)
			return fmt.Errorf("failure creating Deployment: %v", err)
		}

		klog.Infof("Trying to update Deployment: %s for %q as it already existed.", AppName, ingressRequest.managementIngress.Name)
		current := &apps.Deployment{}
		if err = ingressRequest.Get(AppName, ingressRequest.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("failure getting %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		desired, different := utils.IsDeploymentDifferent(current, ds)
		if !different {
			klog.Infof("No change found from the deployment: %s.", AppName)
			return nil
		}
		klog.Infof("Found change from Deployment Replicas %d. Trying to update it.", ingressRequest.managementIngress.Spec.Replicas)

		klog.Infof("Found change from Deployment %+v. Trying to update it.", podSpec)
		err = ingressRequest.Update(desired)
		if err != nil {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failure updating deployment %q: %v", AppName, err)
			return fmt.Errorf("failure updating %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
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

// func (ingressRequest *IngressRequest) waitForDeploymentReady(ds *apps.Deployment) error {

// 	err := wait.Poll(5*time.Second, 2*time.Second, func() (done bool, err error) {
// 		err = ingressRequest.Get(ds.Name, ingressRequest.managementIngress.ObjectMeta.Namespace, ds)
// 		if err != nil {
// 			if errors.IsNotFound(err) {
// 				return false, fmt.Errorf("failed to get Fluentd deployment: %v", err)
// 			}
// 			return false, err
// 		}

// 		if int(ds.Status.ReadyReplicas) == int(ds.Status.Replicas) {
// 			return true, nil
// 		}

// 		return false, nil
// 	})

// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
