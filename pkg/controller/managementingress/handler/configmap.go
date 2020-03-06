package handler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/IBM/ibm-management-ingress-operator/pkg/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/apis/apps"
)

//NewConfigMap stubs an instance of Configmap
func NewConfigMap(name string, namespace string, data map[string]string) *core.ConfigMap {
	return &core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"component": AppName,
			},
		},
		Data: data,
	}
}

func syncConfigmap(ingr *IngressRequest, cm *core.ConfigMap, ingressConfig bool) error {
	utils.AddOwnerRefToObject(cm, utils.AsOwner(ingr.managementIngress))

	klog.Infof("Creating Configmap: %s for %q.", cm.ObjectMeta.Name, ingr.managementIngress.Name)
	err := ingr.Create(cm)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failure creating configmap %q: %v", cm.ObjectMeta.Name, err)
			return fmt.Errorf("Failure creating configmap: %v", err)
		}

		current := &core.ConfigMap{}

		// Update config
		if err = ingr.Get(cm.ObjectMeta.Name, ingr.managementIngress.ObjectMeta.Namespace, current); err != nil {
			return fmt.Errorf("Failure getting Configmap: %q  for %q: %v", cm.ObjectMeta.Name, ingr.managementIngress.Name, err)
		}

		// no data change, just return
		if reflect.DeepEqual(cm.Data, current.Data) {
			return nil
		}

		json, _ := json.Marshal(cm)
		klog.Infof("Configmap was changed to: %s. Try to update the configmap", json)
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

			klog.Infof("Restart management ingress Deployment after config change.")
			ds.Spec.Template.ObjectMeta.Annotations = annotations
			if err := ingr.Update(ds); err != nil {
				ingr.recorder.Eventf(ingr.managementIngress, "Warning", "UpdatedConfigmap", "Failure updating damonset to make it restarted: %v", err)
				klog.Errorf("Failure updating Deployment: %q for %q after config change: %v ", AppName, ingr.managementIngress.Name, err)
			}
		}
	} else {
		ingr.recorder.Eventf(ingr.managementIngress, "Normal", "CreatedConfigmap", "Successfully created or updated configmap %q", AppName)
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

	return nil
}

//RemoveConfigMap with a given name and namespace
func (ingressRequest *IngressRequest) RemoveConfigMap(name string) error {
	configMap := &core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ingressRequest.managementIngress.Namespace,
		},
		Data: map[string]string{},
	}

	klog.Infof("Removing ConfigMap for %q.", ingressRequest.managementIngress.Name)
	err := ingressRequest.Delete(configMap)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failure deleting %q configmap: %v", name, err)
	}

	return nil
}
