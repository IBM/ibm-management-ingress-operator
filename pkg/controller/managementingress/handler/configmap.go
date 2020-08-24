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
	"encoding/json"
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

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
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

// for configmap ibmcloud-cluster-info, need to check whether it's already existed, if so, patch it, else create it
func patchOrCreateConfigmap(ingr *IngressRequest, cm *core.ConfigMap) error {

	err, cfg := ingr.GetConfigmap(ClusterConfigName, ingr.managementIngress.ObjectMeta.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			// create configmap
			klog.Infof("Creating Configmap: %s for %q.", cm.ObjectMeta.Name, ingr.managementIngress.Name)

			err = ingr.Create(cm)
			if err != nil {
				ingr.recorder.Eventf(ingr.managementIngress, "Warning", "CreatedConfigmap", "Failure creating configmap %q: %v", cm.ObjectMeta.Name, err)
				return fmt.Errorf("Failure creating configmap: %v", err)
			}

		} else {
			return err
		}
	} else {
		// need to patch configmap
		if reflect.DeepEqual(cfg.Data, cm.Data) {
			klog.Infof("No change found from the configmap: %s.", cm.ObjectMeta.Name)
			return nil
		}

		var mergePatch []byte
		mergePatch, err := json.Marshal(map[string]interface{}{
			"data": cm.Data,
		})

		if err != nil {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "PatchConfigmap", "Failure encoding patch data for %q: %v", cm.ObjectMeta.Name, err)
			return fmt.Errorf("Failing encoding patch data for %q with value %s: %v", ingr.managementIngress.Name,
				mergePatch, err)
		}
		klog.Infof("Patching Configmap: %s for %q with data %v.", cfg.ObjectMeta.Name, ingr.managementIngress.Name,
			string(mergePatch))

		if err = ingr.Patch(cfg, mergePatch); err != nil {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "PatchConfigmap", "Failure patching configmap for %q: %v", cm.ObjectMeta.Name, err)
			return fmt.Errorf("Failure patching Configmap: %v for %q: %v", cfg.ObjectMeta.Name, ingr.managementIngress.Name, err)
		}
	}

	klog.Infof("Creating or patching configmap succeeded: %v for %q", cm.ObjectMeta.Name, ingr.managementIngress.Name)
	ingr.recorder.Eventf(ingr.managementIngress, "Normal", "CreatedConfigmap", "Successfully created or patched configmap %q", cm.ObjectMeta.Name)
	return nil
}

func syncConfigmap(ingr *IngressRequest, cm *core.ConfigMap, ingressConfig bool) error {
	if err := controllerutil.SetControllerReference(ingr.managementIngress, cm, ingr.scheme); err != nil {
		klog.Errorf("Error setting controller reference on Configmap: %v", err)
	}

	klog.Infof("Creating Configmap: %s for %q.", cm.ObjectMeta.Name, ingr.managementIngress.Name)
	err := ingr.Create(cm)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failure creating configmap %q: %v", cm.ObjectMeta.Name, err)
			return fmt.Errorf("Failure creating configmap: %v", err)
		}

		if !ingressConfig {
			return nil
		}

		klog.Infof("Trying to update Configmap: %s for %q as it already existed.", cm.ObjectMeta.Name, ingr.managementIngress.Name)
		current := &core.ConfigMap{}
		// Update config
		if err = ingr.Get(cm.ObjectMeta.Name, ingr.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("Failure getting Configmap: %q  for %q: %v", cm.ObjectMeta.Name, ingr.managementIngress.Name, err)
		}

		// no data change, just return
		if reflect.DeepEqual(cm.Data, current.Data) {
			klog.Infof("No change found from the configmap: %s.", cm.ObjectMeta.Name)
			return nil
		}

		json, _ := json.Marshal(cm)
		klog.Infof("Found change from Configmap %s. Trying to update it.", json)
		current.Data = cm.Data

		// Apply the latest change to configmap
		if err = ingr.Update(current); err != nil {
			return fmt.Errorf("Failure updating Configmap: %v for %q: %v", cm.ObjectMeta.Name, ingr.managementIngress.Name, err)
		}

		// Restart Deployment because config is updated.
		if ingressConfig {
			ds := &apps.Deployment{}
			if err = ingr.Get(AppName, ingr.managementIngress.ObjectMeta.Namespace, ds); err != nil {
				if !errors.IsNotFound(err) {
					ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failure getting Deployment: %v", err)
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
				ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failure updating damonset to make it restarted: %v", err)
				klog.Errorf("Failure updating Deployment: %q for %q after config change: %v ", AppName, ingr.managementIngress.Name, err)
			}
		}
	} else {
		ingr.recorder.Eventf(ingr.managementIngress, "Normal", "CreatedConfigmap", "Successfully created or updated configmap %q", cm.ObjectMeta.Name)
	}

	return nil
}

func (ingressRequest *IngressRequest) CreateOrUpdateConfigMap() error {

	// Create management ingress config
	config := NewConfigMap(
		ConfigName,
		ingressRequest.managementIngress.Namespace,
		ingressRequest.managementIngress.Spec.Config,
	)

	if err := syncConfigmap(ingressRequest, config, true); err != nil {
		return fmt.Errorf("Failure creating or updating management ingress config for %q: %v", ConfigName, err)
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
		return fmt.Errorf("Failure creating bind info for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	if err := populateCloudClusterInfo(ingressRequest); err != nil {
		return fmt.Errorf("Failure populate cloud cluster info for %q: %v", ingressRequest.managementIngress.Name, err)
	}
	return nil
}

// create configmap ibmcloud-cluster-info
func populateCloudClusterInfo(ingressRequest *IngressRequest) error {

	baseDomain, err := ingressRequest.GetRouteAppDomain()
	if err != nil {
		return fmt.Errorf("Failure getting route base domain %q: %v", ingressRequest.managementIngress.Name, err)
	}

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

	// get api server address and port from configmap console-config in namespace openshift-console
	console := &core.ConfigMap{}
	err = aclient.Get(context.TODO(), types.NamespacedName{Name: ConsoleCfg, Namespace: ConsoleNS}, console)
	if err != nil {
		return err
	}

	var result map[interface{}]interface{}
	var apiaddr string
	yaml.Unmarshal([]byte(console.Data[ConsoleCfgYaml]), &result)

	for k, v := range result {
		if k.(string) == ConsoleClusterInfo {
			cinfo := v.(map[interface{}]interface{})
			for k1, v1 := range cinfo {
				if k1.(string) == ConsoleMasterURL {
					apiaddr = v1.(string)
					if strings.HasPrefix(apiaddr, "https://") {
						apiaddr = apiaddr[8:]
					}
					break
				}
			}
			break
		}
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
		},
	)

	if err := patchOrCreateConfigmap(ingressRequest, clustercfg); err != nil {
		return fmt.Errorf("Failure creating cluster info for %q: %v", ingressRequest.managementIngress.Name, err)
	}

	return nil
}
