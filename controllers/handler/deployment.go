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
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/IBM/ibm-management-ingress-operator/utils"
)

const (
	httpsPort = int32(8443)
	httpPort  = int32(8080)
)

//NewDeployment stubs an instance of a deployment
func NewDeployment(name string, namespace string, replicas int32, podSpec core.PodSpec) *apps.Deployment {

	labels := GetCommonLabels()
	podLabels := GetCommonLabels()
	commAnnotations := GetCommonAnnotations()
	podAnnotations := map[string]string{
		"scheduler.alpha.kubernetes.io/critical-pod": "",
		"clusterhealth.ibm.com/dependencies":         "cert-manager, auth-idp",
	}

	// Merget common annotations with pod specific annotations.
	for k, v := range commAnnotations {
		podAnnotations[k] = v
	}

	// add label for namespace operator
	podLabels["intent"] = "projected"

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
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
		},
	}
}

func newPodSpec(img, clusterDomain string, resources *core.ResourceRequirements, nodeSelector map[string]string,
	tolerations []core.Toleration, allowedHostHeader string, fipsEnabled bool) core.PodSpec {
	namespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		klog.Error("failure getting watch namespace")
		os.Exit(1)
	}
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
		"--configmap=$(POD_NAMESPACE)/" + ConfigName,
		"--http-port=8080",
		"--https-port=8443",
	}

	if len(namespace) > 0 {
		container.Command = append(container.Command, "--watch-namespace=$(WATCH_NAMESPACE)")
	}

	container.Env = []core.EnvVar{
		{Name: "WATCH_NAMESPACE", ValueFrom: &core.EnvVarSource{
			ConfigMapKeyRef: &core.ConfigMapKeySelector{
				Key: "namespaces",
				LocalObjectReference: core.LocalObjectReference{
					Name: NamespaceScopeConfigMap}}}},
		{Name: "ENABLE_IMPERSONATION", Value: "false"},
		{Name: "APISERVER_SECURE_PORT", Value: "6443"},
		{Name: "CLUSTER_DOMAIN", Value: clusterDomain},
		{Name: "HOST_HEADERS_CHECK_ENABLED", Value: "false"},
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

	spreadConstraints := []core.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/zone",
			WhenUnsatisfiable: core.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": AppName,
				},
			},
		},
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/region",
			WhenUnsatisfiable: core.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": AppName,
				},
			},
		},
	}

	podSpec := core.PodSpec{
		Containers:                []core.Container{container},
		ServiceAccountName:        ServiceAccountName,
		NodeSelector:              nodeSelector,
		Tolerations:               tolerations,
		Affinity:                  affinity,
		TopologySpreadConstraints: spreadConstraints,
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

func getClusterDomain(clusterType string) (string, error) {
	if clusterType == CNCF {
		return "cluster.local", nil
	}

	dns := &operatorv1.DNS{}
	clusterClient, err := createOrGetClusterClient()
	if err != nil {
		return "", fmt.Errorf("failure creating or getting cluster client: %v", err)
	}
	if err := clusterClient.Get(context.TODO(), types.NamespacedName{Name: "default", Namespace: ""}, dns); err != nil {
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

func (ingressRequest *IngressRequest) CreateOrUpdateDeployment(clusterType string) error {
	image := os.Getenv("ICP_MANAGEMENT_INGRESS_IMAGE")

        var hostHeader string
	if clusterType == CNCF {
		pos := strings.LastIndex(ingressRequest.managementIngress.Status.Host, ":")
		dn := ingressRequest.managementIngress.Status.Host[0:pos]
		hostHeader = strings.Join([]string{
			ingressRequest.managementIngress.Spec.AllowedHostHeader,
			dn,
			ServiceName,
			IAMTokenService,
			strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace}, "."),
			strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace, "svc"}, "."),
			strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace}, "."),
			strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace, "svc"}, "."),
		}, " ")
	} else {
		hostHeader = strings.Join([]string{
			ingressRequest.managementIngress.Spec.AllowedHostHeader,
			ingressRequest.managementIngress.Status.Host,
			ServiceName,
			IAMTokenService,
			strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace}, "."),
			strings.Join([]string{ServiceName, ingressRequest.managementIngress.Namespace, "svc"}, "."),
			strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace}, "."),
			strings.Join([]string{IAMTokenService, ingressRequest.managementIngress.Namespace, "svc"}, "."),
		}, " ")
	}

	clusterDomain, err := getClusterDomain(clusterType)

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
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failed to create deployment %q", AppName)
			return fmt.Errorf("failure creating Deployment: %v", err)
		}

		klog.Infof("Trying to update Deployment: %s as it already existed.", AppName)
		current := &apps.Deployment{}
		if err = ingressRequest.Get(AppName, ingressRequest.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("failure getting %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		desired, different := utils.IsDeploymentDifferent(current, ds)
		if !different {
			klog.Infof("No change found from the deployment: %s, skip updating current deployment.", AppName)
			return nil
		}
		klog.Infof("Found change for deployment: %s, trying to update it.", AppName)
		err = ingressRequest.Update(desired)
		if err != nil {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedDeployment", "Failed to update deployment: %s", AppName)
			return fmt.Errorf("failure updating %q Deployment for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "UpdatedDeployment", "Successfully updated deployment %q", AppName)
		return nil
	}

	klog.Infof("Created Deployment: %s.", AppName)
	ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedDeployment", "Successfully created deployment %q", AppName)

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
