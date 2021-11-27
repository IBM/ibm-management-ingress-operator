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
	operatorv1 "github.com/openshift/api/operator/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

var (
	clusterClient               client.Client
	scheme                      = runtime.NewScheme()
	ConfigMapSchemeGroupVersion = schema.GroupVersion{Group: "", Version: "v1"}
	OperatorSchemeGroupVersion  = schema.GroupVersion{Group: "operator.openshift.io", Version: "v1"}
)

func createOrGetClusterClient() (client.Client, error) {
	// return if cluster client already exists
	if clusterClient != nil {
		return clusterClient, nil
	}
	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	scheme.AddKnownTypes(ConfigMapSchemeGroupVersion, &core.ConfigMap{}, &core.ConfigMapList{})
	scheme.AddKnownTypes(OperatorSchemeGroupVersion, &operatorv1.IngressController{}, &operatorv1.IngressControllerList{})
	scheme.AddKnownTypes(OperatorSchemeGroupVersion, &operatorv1.DNS{}, &operatorv1.DNSList{})

	clusterClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return clusterClient, nil
}

func waitForSecret(r *IngressRequest, name string, stopCh <-chan struct{}) (*core.Secret, error) {
	klog.Infof("Waiting for secret: %s ...", name)
	s := &core.Secret{}

	err := wait.PollImmediateUntil(2*time.Second, func() (done bool, err error) {
		if err := r.Get(name, r.managementIngress.ObjectMeta.Namespace, s); err != nil {
			return false, nil
		}
		return true, nil
	}, stopCh)

	return s, err
}
