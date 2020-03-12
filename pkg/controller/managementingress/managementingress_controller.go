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

	err = k8shandler.Reconcile(instance, r.client, r.recorder)
	if err != nil {
		klog.Errorf("Failure reconciling ManagementIngress: %v", err)
		return reconcile.Result{}, err
	} else {
		klog.Infof("Reconciling ManagementIngress was successful")
	}
	return reconcile.Result{}, nil
}
