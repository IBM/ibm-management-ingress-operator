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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
	k8shandler "github.com/IBM/ibm-management-ingress-operator/pkg/controller/managementingress/handler"
	k8sutils "github.com/IBM/ibm-management-ingress-operator/pkg/utils"
)

const (
	controllerName = "managementingress-controller"
)

// Add creates a new ManagementIngress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileManagementIngress{
		client:   mgr.GetClient(),
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

	return nil
}

// blank assignment to verify that ReconcileManagementIngress implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileManagementIngress{}

// ReconcileManagementIngress reconciles a ManagementIngress object
type ReconcileManagementIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileManagementIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling ManagementIngress")

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

	i := k8shandler.NewIngressHandler(instance, r.client, r.recorder, r.scheme)

	finalizerName := "managementIngress-clean-up"

	// examine DeletionTimestamp to determine if object is under deletion
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !k8sutils.ContainsString(instance.ObjectMeta.Finalizers, finalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
			if err := r.client.Update(context.Background(), instance); err != nil {
				klog.Errorf("Failed to add finalizer: ", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if k8sutils.ContainsString(instance.ObjectMeta.Finalizers, finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := k8shandler.DeleteClusterResources(i); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				klog.Errorf("Failed to delete cluster resources: ", err)
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = k8sutils.RemoveString(instance.ObjectMeta.Finalizers, finalizerName)
			if err := r.client.Update(context.Background(), instance); err != nil {
				klog.Errorf("Failed to remove finalizer: ", err)
				return reconcile.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return reconcile.Result{}, nil
	}

	err = k8shandler.Reconcile(i)
	if err != nil {
		klog.Errorf("Failure reconciling ManagementIngress: %v", err)
		return reconcile.Result{}, err
	} else {
		klog.Infof("Reconciling ManagementIngress was successful")
	}
	return reconcile.Result{}, nil
}
