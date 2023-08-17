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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	labels "k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/IBM/controller-filtered-cache/filteredcache"
	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
	"github.com/IBM/ibm-management-ingress-operator/controllers"
	"github.com/IBM/ibm-management-ingress-operator/controllers/handler"
	"github.com/IBM/ibm-management-ingress-operator/version"
)

var (
	operatorSDKVersion = "v1.1.0"
	operatorName       = "ibm-management-ingress-operator"
	// scheme             = k8sruntime.NewScheme()
)

func printVersion() {
	klog.Infof(fmt.Sprintf("Operator Version: %s", version.Version))
	klog.Infof(fmt.Sprintf("Go Version: %s", runtime.Version()))
	klog.Infof(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	klog.Infof(fmt.Sprintf("Version of operator-sdk: %s", operatorSDKVersion))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8383", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.Parse()

	printVersion()

	watchNS, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		klog.Error("failure getting watch namespace")
		os.Exit(1)
	}

	operatorNs, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		klog.Error("failure getting operator namespace")
		os.Exit(1)
	}

	commonLabels := handler.GetCommonLabels()
	labelSelector := labels.SelectorFromSet(commonLabels).String()

	gvkLabelMap := map[schema.GroupVersionKind]filteredcache.Selector{
		corev1.SchemeGroupVersion.WithKind("ConfigMap"): {
			LabelSelector: labelSelector,
		},
		appsv1.SchemeGroupVersion.WithKind("Deployment"): {
			LabelSelector: labelSelector,
		},
		corev1.SchemeGroupVersion.WithKind("Service"): {
			LabelSelector: labelSelector,
		},
		corev1.SchemeGroupVersion.WithKind("Secret"): {
			LabelSelector: labelSelector,
		},
	}

	scheme := k8sruntime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(certmanagerv1alpha1.AddToScheme(scheme))

	var ctrlOpt ctrl.Options
	if strings.Contains(watchNS, ",") {
		namespaces := strings.Split(watchNS, ",")
		// Create MultiNamespacedCache with watched namespaces if the watch namespace string contains comma
		ctrlOpt = ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     metricsAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       operatorName,
			NewCache:               filteredcache.MultiNamespacedFilteredCacheBuilder(gvkLabelMap, namespaces),
			HealthProbeBindAddress: probeAddr,
		}
	} else {
		// Create manager option for watching all namespaces.
		ctrlOpt = ctrl.Options{
			Scheme:                 scheme,
			Namespace:              watchNS,
			MetricsBindAddress:     metricsAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       operatorName,
			NewCache:               filteredcache.NewFilteredCacheBuilder(gvkLabelMap),
			HealthProbeBindAddress: probeAddr,
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOpt)

	if err != nil {
		klog.Errorf("unable to start manager: %v", err)
		os.Exit(1)
	}

	var clusterType string
	var domainName string
	// var nodePort string
	ibmCppConfig := &corev1.ConfigMap{}
	if err := mgr.GetClient().Get(context.TODO(), types.NamespacedName{Name: handler.CppConfigName, Namespace: operatorNs}, ibmCppConfig); !errors.IsNotFound(err) {
		utilruntime.Must(routev1.AddToScheme(scheme))
		ctrlOpt.Scheme = scheme
		mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOpt)
		if err != nil {
			klog.Errorf("unable to start manager: %v", err)
			os.Exit(1)
		} else {
			clusterType = ibmCppConfig.Data[handler.KubernetesClusterType]
			domainName = ibmCppConfig.Data[handler.CppConfigDomainName]
			// dns = projectkConfig.Data["dns"]
		}
	} else if err != nil {
		klog.Errorf("unable to start manager because %s configmap not found in ns %s: %v", handler.CppConfigName, operatorNs, err)
		os.Exit(1)
	}

	if err = (&controllers.ManagementIngressReconciler{
		Client:      mgr.GetClient(),
		Reader:      mgr.GetAPIReader(),
		Scheme:      mgr.GetScheme(),
		Recorder:    mgr.GetEventRecorderFor(controllers.ControllerName),
		ClusterType: clusterType,
		DomainName:  domainName,
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to create controller: %v", err)
		os.Exit(1)
	}

	klog.Info("Setting up liveness and readiness probes")
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		klog.Errorf("unable to set up health check: %v", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		klog.Errorf("unable to set up ready check: %v", err)
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("problem running manager: %v", err)
		os.Exit(1)
	}
}
