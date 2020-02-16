package handler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/apis/apps"

	"github.com/IBM/management-ingress-operator/pkg/utils"
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

func (ingressRequest *IngressRequest) CreateOrUpdateConfigMap() error {
	configmap := NewConfigMap(
		AppName,
		ingressRequest.managementIngress.Namespace,
		ingressRequest.managementIngress.Spec.Config,
	)

	utils.AddOwnerRefToObject(configmap, utils.AsOwner(ingressRequest.managementIngress))

	klog.Infof("Creating Configmap for %q.", ingressRequest.managementIngress.Name)
	err := ingressRequest.Create(configmap)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedConfigmap", "Failure creating configmap %q: %v", AppName, err)
			return fmt.Errorf("Failure creating configmap: %v", err)
		}

		current := &core.ConfigMap{}

		// Update config
		if err = ingressRequest.Get(AppName, current); err != nil {
			return fmt.Errorf("Failure getting %q configmap for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		if reflect.DeepEqual(configmap.Data, current.Data) {
			return nil
		}

		json, _ := json.Marshal(configmap)
		klog.Infof("Configmap was changed to: %s. Try to update the configmap", json)
		current.Data = configmap.Data

		if err = ingressRequest.Update(current); err != nil {
			return fmt.Errorf("Failure updating %v configmap for %q: %v", AppName, ingressRequest.managementIngress.Name, err)
		}

		// Restart Deployment because config is updated.
		ds := &apps.Deployment{}
		if err = ingressRequest.Get(AppName, ds); err != nil {
			if !errors.IsNotFound(err) {
				ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedConfigmap", "Failure getting Deployment: %v", err)
				klog.Errorf("Failure getting %q Deployment for %q after config change: %v ", AppName, ingressRequest.managementIngress.Name, err)
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
		if err := ingressRequest.Update(ds); err != nil {
			ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Warning", "UpdatedConfigmap", "Failure updating damonset to make it restarted: %v", err)
			klog.Errorf("Failure updating %q Deployment for %q after config change: %v ", AppName, ingressRequest.managementIngress.Name, err)
		}
	} else {
		ingressRequest.recorder.Eventf(ingressRequest.managementIngress, "Normal", "CreatedConfigmap", "Successfully created or updated configmap %q", AppName)
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
