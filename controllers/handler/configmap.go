//
// Copyright 2021 IBM Corporation
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
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/IBM/ibm-management-ingress-operator/utils"
)

//NewConfigMap stubs an instance of Configmap
func NewConfigMap(name string, namespace string, data map[string]string) *core.ConfigMap {

	labels := GetCommonLabels()

	return &core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}

// for configmap ibmcloud-cluster-info, need to check whether it's already existed, if so, update it, else create it
func updateClusterInfo(ingr *IngressRequest, cm *core.ConfigMap) error {
	if err := controllerutil.SetControllerReference(ingr.managementIngress, cm, ingr.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Configmap: %v", err)
	}

	cfg, err := ingr.GetConfigmap(ClusterConfigName, ingr.managementIngress.ObjectMeta.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			// create configmap
			err = ingr.Create(cm)
			if err != nil {
				ingr.recorder.Eventf(ingr.managementIngress, "Warning", "CreatedConfigmap", "Failed to create configmap: %s", cm.ObjectMeta.Name)
				return fmt.Errorf("failure creating configmap: %v", err)
			}
			klog.Infof("Created configmap %s.", cm.ObjectMeta.Name)
			ingr.recorder.Eventf(ingr.managementIngress, "Normal", "CreatedConfigmap", "Successfully created configmap: %s", cm.ObjectMeta.Name)
			return nil
		}
	} else {
		klog.Infof("Trying to update configmap: %s as it already existed.", cfg.ObjectMeta.Name)
		if reflect.DeepEqual(cm.Data, cfg.Data) {
			klog.Infof("No change found from the configmap: %s, skip updating current configmap.", cm.ObjectMeta.Name)
			return nil
		}

		klog.Infof("Found change for configmap %s, trying to update it.", cfg.ObjectMeta.Name)
		cfg.Data = cm.Data
		if err := ingr.Update(cfg); err != nil {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failed to update configmap: %s", cm.ObjectMeta.Name)
			return fmt.Errorf("failure updating Configmap %s: %v", cfg.ObjectMeta.Name, err)
		}
		ingr.recorder.Eventf(ingr.managementIngress, "Normal", "UpdatedConfigmap", "Successfully updated configmap: %s", cm.ObjectMeta.Name)
	}

	return err
}

func syncConfigmap(ingr *IngressRequest, cm *core.ConfigMap, ingressConfig bool) error {
	if err := controllerutil.SetControllerReference(ingr.managementIngress, cm, ingr.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Configmap: %v", err)
	}

	err := ingr.Create(cm)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "CreatedConfigmap", "Failed to create configmap: %s", cm.ObjectMeta.Name)
			return fmt.Errorf("failure creating configmap: %v", err)
		}

		klog.Infof("Trying to update configmap: %s as it already existed.", cm.ObjectMeta.Name)
		current := &core.ConfigMap{}
		// Update config
		if err = ingr.Get(cm.ObjectMeta.Name, ingr.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("failure getting Configmap: %q: %v", cm.ObjectMeta.Name, err)
		}

		// no data change, just return
		if reflect.DeepEqual(cm.Data, current.Data) {
			klog.Infof("No change found from the configmap: %s, skip updating current configmap.", cm.ObjectMeta.Name)
			return nil
		}

		//json, _ := json.Marshal(cm)
		klog.Infof("Found change for configmap %s, trying to update it.", cm.ObjectMeta.Name)
		current.Data = cm.Data

		// Apply the latest change to configmap
		if err = ingr.Update(current); err != nil {
			return fmt.Errorf("failure updating Configmap: %v: %v", cm.ObjectMeta.Name, err)
		}

		// Restart Deployment because management-ingress-config is updated.
		if ingressConfig {
			ds := &apps.Deployment{}
			if err = ingr.Get(AppName, ingr.managementIngress.ObjectMeta.Namespace, ds); err != nil {
				if !errors.IsNotFound(err) {
					klog.Errorf("Failure getting Deployment: %q for %q after config change: %v ", AppName, ingr.managementIngress.Name, err)
				}
				return nil
			}

			annotations := ds.Spec.Template.ObjectMeta.Annotations
			// Update annotation to force restart of ds pods
			annotations = utils.AppendAnnotations(
				annotations,
				map[string]string{
					ConfigUpdateAnnotationKey: time.Now().Format(time.RFC850),
				},
			)

			klog.Infof("Restarting management ingress Deployment after config change.")
			ds.Spec.Template.ObjectMeta.Annotations = annotations
			if err := ingr.Update(ds); err != nil {
				klog.Errorf("Failure updating Deployment: %q after config change: %v ", AppName, err)
			}
		}
	} else {
		klog.Infof("Created Configmap: %s.", cm.ObjectMeta.Name)
		ingr.recorder.Eventf(ingr.managementIngress, "Normal", "CreatedConfigmap", "Successfully created or updated configmap %q", cm.ObjectMeta.Name)
	}

	return nil
}

func (ingressRequest *IngressRequest) CreateOrUpdateConfigMap(clusterType string, domainName string) error {

	// Create management ingress config
	config := NewConfigMap(
		ConfigName,
		ingressRequest.managementIngress.Namespace,
		ingressRequest.managementIngress.Spec.Config,
	)

	if err := syncConfigmap(ingressRequest, config, true); err != nil {
		return fmt.Errorf("failure creating or updating management ingress config for %q: %v", ConfigName, err)
	}

	// Create bindinfo
	bindInfo := NewConfigMap(
		BindInfoConfigMap,
		ingressRequest.managementIngress.Namespace,
		map[string]string{
			"MANAGEMENT_INGRESS_ROUTE_HOST":   ingressRequest.managementIngress.Status.Host,
			"MANAGEMENT_INGRESS_SERVICE_NAME": ServiceName,
		},
	)

	if err := syncConfigmap(ingressRequest, bindInfo, false); err != nil {
		return fmt.Errorf("failure creating bind info for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// Create ibmcloud-cluster-info
	if err := populateCloudClusterInfo(ingressRequest, clusterType, domainName); err != nil {
		return fmt.Errorf("failure populate cloud cluster info for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}

// create configmap ibmcloud-cluster-info
func populateCloudClusterInfo(ingressRequest *IngressRequest, clusterType string, domainName string) error {
	ver := os.Getenv(CSVersionEnv)
	if ver == "" {
		ver = CSVersionValue
	}
	cname := os.Getenv(ClusterNameEnv)
	if cname == "" {
		cname = ClusterNameValue
	}

	rhttpPort := os.Getenv(RouteHTTPPortEnv)
	if rhttpPort == "" {
		rhttpPort = RouteHTTPPortValue
	}

	rhttpsPort := os.Getenv(RouteHTTPSPortEnv)
	if rhttpsPort == "" {
		rhttpsPort = RouteHTTPSPortValue
	}

	ns := os.Getenv(PODNAMESPACE)
	ep := "https://" + ServiceName + "." + ns + ".svc:443"
	if clusterType == CNCF {
		pos := strings.LastIndex(domainName, ":")
		// dn is domain name without nodeport
		dn := domainName[0:pos]
		node_port := domainName[pos+1:]
		cncfDomainName := strings.Join([]string{ConsoleRouteName, dn}, ".")
		clustercfg := NewConfigMap(
			ClusterConfigName,
			ingressRequest.managementIngress.Namespace,
			map[string]string{
				ClusterAddr:     cncfDomainName,
				ClusterCADomain: cncfDomainName,
				ClusterEP:       ep,
				ClusterName:     cname,
				RouteHTTPPort:   rhttpPort,
				RouteHTTPSPort:  rhttpsPort,
				CSVersion:       ver,
				ProxyAddress:    cncfDomainName,
				NodePort:        node_port,
				ProxyHTTPPort:   "80",
				ProxyHTTPSPort:  "443",
			},
		)
		if err := updateClusterInfo(ingressRequest, clustercfg); err != nil {
			return fmt.Errorf("failure creating cluster info for %q: %v", ingressRequest.managementIngress.Name, err)
		}
		return nil
	}
	baseDomain, err := ingressRequest.GetRouteAppDomain()
	if err != nil {
		return fmt.Errorf("failure getting route base domain %q: %v", ingressRequest.managementIngress.Name, err)
	}

	// get api server address and port from configmap console-config in namespace openshift-console
	console := &core.ConfigMap{}
	clusterClient, err := createOrGetClusterClient()
	if err != nil {
		return fmt.Errorf("failure creating or getting cluster client: %v", err)
	}
	err = clusterClient.Get(context.TODO(), types.NamespacedName{Name: ConsoleCfg, Namespace: ConsoleNS}, console)
	if err != nil {
		return err
	}

	var result map[interface{}]interface{}
	var apiaddr string
	if err = yaml.Unmarshal([]byte(console.Data[ConsoleCfgYaml]), &result); err != nil {
		return err
	}

	for k, v := range result {
		if k.(string) == ConsoleClusterInfo {
			cinfo := v.(map[interface{}]interface{})
			for k1, v1 := range cinfo {
				if k1.(string) == ConsoleMasterURL {
					apiaddr = v1.(string)
					apiaddr = strings.TrimPrefix(apiaddr, "https://")
					break
				}
			}
			break
		}
	}

	proxyRouteHost, err := ingressRequest.GetProxyRouteHost()
	if err != nil {
		return fmt.Errorf("failure getting proxy route host: %v", err)
	}

	pos := strings.LastIndex(apiaddr, ":")

	clustercfg := NewConfigMap(
		ClusterConfigName,
		ingressRequest.managementIngress.Namespace,
		map[string]string{
			ClusterAddr:          ingressRequest.managementIngress.Status.Host,
			ClusterCADomain:      ingressRequest.managementIngress.Status.Host,
			ClusterEP:            ep,
			ClusterName:          cname,
			RouteHTTPPort:        rhttpPort,
			RouteHTTPSPort:       rhttpsPort,
			RouteBaseDomain:      baseDomain,
			CSVersion:            ver,
			ClusterAPIServerHost: apiaddr[0:pos],
			ClusterAPIServerPort: apiaddr[pos+1:],
			ProxyAddress:         proxyRouteHost,
			ProxyHTTPPort:        "80",
			ProxyHTTPSPort:       "443",
		},
	)

	if err := updateClusterInfo(ingressRequest, clustercfg); err != nil {
		return fmt.Errorf("failure creating cluster info for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}
