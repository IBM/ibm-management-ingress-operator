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
package controllers

import (
	"context"

	certmanagerv1alpha1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
	k8shandler "github.com/IBM/ibm-management-ingress-operator/controllers/handler"
)

const (
	ControllerName = "managementingress-controller"
)

// ManagementIngressReconciler reconciles a ManagementIngress object
type ManagementIngressReconciler struct {
	client.Client
	Reader      client.Reader
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	ClusterType string
	DomainName  string
}

// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ManagementIngressReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	// Fetch the ManagementIngress instance
	managementingress := &operatorv1alpha1.ManagementIngress{}
	err := r.Get(ctx, request.NamespacedName, managementingress)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("managementingress: %s/%s not found. Ignoring since object must be deleted", request.NamespacedName.Namespace, request.NamespacedName.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("failed to get managementingress: %s/%s", request.NamespacedName.Namespace, request.NamespacedName.Name)
		return ctrl.Result{}, err
	}

	if managementingress.Spec.ManagementState == operatorv1alpha1.ManagementStateUnmanaged {
		klog.Errorf("do nothing for the managementingress: %s/%s because its state is unmanaged", request.NamespacedName.Namespace, request.NamespacedName.Name)
		return ctrl.Result{}, nil
	}

	if !managementingress.ObjectMeta.DeletionTimestamp.IsZero() {
		klog.Infof("do nothing for the managementingress: %s/%s because it was deleted", request.NamespacedName.Namespace, request.NamespacedName.Name)
		return ctrl.Result{}, nil
	}

	klog.Infof("reconciling managementingress: %s/%s", request.NamespacedName.Namespace, request.NamespacedName.Name)
	ingresshandler := k8shandler.NewIngressHandler(managementingress, r.Client, r.Recorder, r.Scheme)

	requeue, err := k8shandler.Reconcile(ingresshandler, r.ClusterType, r.DomainName)
	if err == nil && requeue {
		klog.Infof("Requeueing for upgrade deployment delete")
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		klog.Errorf("failed to reconcile managementingress: %s/%s with error: %v", request.NamespacedName.Namespace, request.NamespacedName.Name, err)
		return ctrl.Result{}, err
	}
	klog.Infof("reconciling managementingress: %s/%s was successful", request.NamespacedName.Namespace, request.NamespacedName.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager set up a new controller that will be started by the provided manager.
func (r *ManagementIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.ClusterType == "cncf" {
		return ctrl.NewControllerManagedBy(mgr).
			For(&operatorv1alpha1.ManagementIngress{}).
			Owns(&corev1.Service{}).
			Owns(&corev1.Secret{}).
			Owns(&corev1.ConfigMap{}).
			Owns(&corev1.ServiceAccount{}).
			Owns(&appsv1.Deployment{}).
			Owns(&certmanagerv1alpha1.Certificate{}).
			Complete(r)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.ManagementIngress{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&certmanagerv1alpha1.Certificate{}).
		Owns(&routev1.Route{}).
		Complete(r)
}
