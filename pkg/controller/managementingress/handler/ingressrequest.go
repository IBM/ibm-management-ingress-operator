package handler

import (
	"context"
	"time"

	v1alpha1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressRequest struct {
	client            client.Client
	managementIngress *v1alpha1.ManagementIngress
	recorder          record.EventRecorder
}

func (ingressRequest *IngressRequest) isManaged() bool {
	return ingressRequest.managementIngress.Spec.ManagementState == v1alpha1.ManagementStateManaged
}

func (ingressRequest *IngressRequest) Create(object runtime.Object) (err error) {
	if err = ingressRequest.client.Create(context.TODO(), object); err != nil {
		klog.Errorf("Error creating %v: %v", object.GetObjectKind(), err)
	}
	return err
}

//Update the runtime Object or return error
func (ingressRequest *IngressRequest) Update(object runtime.Object) (err error) {
	if err = ingressRequest.client.Update(context.TODO(), object); err != nil {
		klog.Errorf("Error updating %v: %v", object.GetObjectKind(), err)
	}
	return err
}

//Update the runtime Object status or return error
func (ingressRequest *IngressRequest) UpdateStatus(object runtime.Object) (err error) {
	if err = ingressRequest.client.Status().Update(context.TODO(), object); err != nil {
		// making this debug because we should be throwing the returned error if we are never
		// able to update the status
		klog.V(4).Infof("Error updating status: %v", err)
	}
	return err
}

func (ingressRequest *IngressRequest) Get(objectName, objectNamespace string, object runtime.Object) error {
	namespace := types.NamespacedName{Name: objectName, Namespace: objectNamespace}
	klog.V(4).Infof("Getting namespace: %v, object: %v", namespace, object)

	return ingressRequest.client.Get(context.TODO(), namespace, object)
}

func (ingressRequest *IngressRequest) List(selector map[string]string, object runtime.Object) error {
	klog.V(4).Infof("Listing selector: %v, object: %v", selector, object)
	labelSelector := labels.SelectorFromSet(selector)

	return ingressRequest.client.List(
		context.TODO(),
		object,
		&client.ListOptions{Namespace: ingressRequest.managementIngress.ObjectMeta.Namespace, LabelSelector: labelSelector},
	)
}

func (ingressRequest *IngressRequest) GetSecret(name string) (error, *core.Secret) {
	secret := &core.Secret{}

	err := wait.Poll(3*time.Second, 2*time.Second, func() (done bool, err error) {
		err = ingressRequest.Get(name, ingressRequest.managementIngress.ObjectMeta.Namespace, secret)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return err, nil
	}

	return nil, secret
}

func (ingressRequest *IngressRequest) Delete(object runtime.Object) (err error) {
	if err = ingressRequest.client.Delete(context.TODO(), object); err != nil {
		klog.V(4).Infof("Error updating status: %v", err)
	}

	return err
}

func GetCommonLabels() map[string]string {
	return map[string]string{
		"app":                          AppName,
		"app.kubernetes.io/component":  AppName,
		"app.kubernetes.io/name":       AppName,
		"app.kubernetes.io/instance":   ServiceName,
		"app.kubernetes.io/managed-by": "",
	}
}

func GetCommonAnnotations() map[string]string {
	return map[string]string{
		"productName":    ProductName,
		"productID":      ProductID,
		"productVersion": ProductVersion,
		"productMetric":  ProductMetric,
	}
}
