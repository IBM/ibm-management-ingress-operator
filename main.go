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
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	certmanagerv1alpha1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/IBM/ibm-management-ingress-operator/api/v1alpha1"
	"github.com/IBM/ibm-management-ingress-operator/controllers"
	"github.com/IBM/ibm-management-ingress-operator/version"
)

var (
	operatorSDKVersion = "v1.1.0"
	operatorName       = "ibm-management-ingress-operator"
	scheme             = k8sruntime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(certmanagerv1alpha1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
}

func printVersion() {
	klog.Infof(fmt.Sprintf("Operator Version: %s", version.Version))
	klog.Infof(fmt.Sprintf("Go Version: %s", runtime.Version()))
	klog.Infof(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	klog.Infof(fmt.Sprintf("Version of operator-sdk: %s", operatorSDKVersion))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8383", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	printVersion()

	ns, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		klog.Error("failure getting watch namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		Namespace:          ns,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   operatorName,
	})

	if err != nil {
		klog.Errorf("unable to start manager: %v", err)
		os.Exit(1)
	}

	if err = (&controllers.ManagementIngressReconciler{
		Client:   mgr.GetClient(),
		Reader:   mgr.GetAPIReader(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(controllers.ControllerName),
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to create controller: %v", err)
		os.Exit(1)
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("problem running manager: %v", err)
		os.Exit(1)
	}
}
