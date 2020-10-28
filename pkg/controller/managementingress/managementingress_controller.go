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
package managementingress

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
	k8shandler "github.com/IBM/ibm-management-ingress-operator/pkg/controller/managementingress/handler"
)

const (
	controllerName = "managementingress-controller"
)

var (
	// watchedResources contains all resources we will watch and reconcile when changed
	// Ideally this would also contain Istio CRDs, but there is a race condition here - we cannot watch
	// a type that does not yet exist.
	watchedResources = []schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "", Version: "v1", Kind: "Service"},
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "", Version: "v1", Kind: "ServiceAccount"},
		{Group: "certmanager.k8s.io", Version: "v1alpha1", Kind: "Certificate"},
		{Group: "route.openshift.io", Version: "v1", Kind: "Route"},
	}

	ownedResourcePredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// only handle delete event in case user accidentally removed the managed resource.
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}
)

var (
	ConfigMapSchemeGroupVersion = schema.GroupVersion{Group: "", Version: "v1"}
	OperatorSchemeGroupVersion  = schema.GroupVersion{Group: "operator.openshift.io", Version: "v1"}
)

// Add creates a new ManagementIngress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	// Get a config to talk to the apiserver
	cfg := mgr.GetConfig()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(ConfigMapSchemeGroupVersion, &core.ConfigMap{}, &core.ConfigMapList{})
	scheme.AddKnownTypes(OperatorSchemeGroupVersion, &operatorv1.IngressController{}, &operatorv1.IngressControllerList{})

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil
	}

	return &ReconcileManagementIngress{
		eClient:  c,
		client:   mgr.GetClient(),
		reader:   mgr.GetAPIReader(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(controllerName),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ManagementIngress
	err = c.Watch(&source.Kind{Type: &v1alpha1.ManagementIngress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	//watch for changes to operand resources
	err = watchOperandResources(c)
	if err != nil {
		return err
	}

	klog.Info("Controller added")
	return nil
}

// blank assignment to verify that ReconcileManagementIngress implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileManagementIngress{}

// ReconcileManagementIngress reconciles a ManagementIngress object
type ReconcileManagementIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	eClient  client.Client
	client   client.Client
	reader   client.Reader
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileManagementIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Fetch the ManagementIngress instance
	instance := &v1alpha1.ManagementIngress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Spec.ManagementState == v1alpha1.ManagementStateUnmanaged {
		return reconcile.Result{}, nil
	}

	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		klog.Infof("Instance: %s/%s was deleted. Do nothing!", request.NamespacedName.Namespace, request.NamespacedName.Name)
		return reconcile.Result{}, nil
	}

	klog.Info("Reconciling ManagementIngress")
	i := k8shandler.NewIngressHandler(instance, r.client, r.eClient, r.recorder, r.scheme)

	err = k8shandler.Reconcile(i)
	if err != nil {
		klog.Errorf("Failure reconciling ManagementIngress: %v", err)
		return reconcile.Result{}, err
	}
	klog.Infof("Reconciling ManagementIngress was successful")
	return reconcile.Result{}, nil
}

// Watch changes for Istio resources managed by the operator
func watchOperandResources(c controller.Controller) error {
	for _, t := range watchedResources {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    t.Kind,
			Group:   t.Group,
			Version: t.Version,
		})
		err := c.Watch(&source.Kind{Type: u}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.ManagementIngress{},
		}, ownedResourcePredicates)

		if err != nil {
			klog.Errorf("Could not create watch for %s/%s/%s: %s.", t.Kind, t.Group, t.Version, err)
		}
	}
	return nil
}
