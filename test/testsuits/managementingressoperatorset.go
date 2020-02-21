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

package testsuits

import (
	"testing"

	olmclient "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/IBM/ibm-management-ingress-operator/test/helpers"
)

// ManagementIngressOperatorSetCreate is for testing the create of the ManagementIngressOperatorSet
func ManagementIngressOperatorSetCreate(t *testing.T) {

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global
	olmClient, err := olmclient.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err = helpers.CreateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// ManagementIngressOperatorSetCreateUpdate is for testing the create and update of the ManagementIngressOperatorSet
func ManagementIngressOperatorSetCreateUpdate(t *testing.T) {

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global
	olmClient, err := olmclient.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err = helpers.CreateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}

	if err = helpers.UpdateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// ManagementIngressOperatorSetCreateDelete is for testing the create and delete of the ManagementIngressOperatorSet
func ManagementIngressOperatorSetCreateDelete(t *testing.T) {

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global
	olmClient, err := olmclient.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err = helpers.CreateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}

	if err = helpers.DeleteTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// ManagementIngressOperatorConfigUpdate is for testing the update of the ManagementIngressOperatorConfig
func ManagementIngressOperatorConfigUpdate(t *testing.T) {

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global
	olmClient, err := olmclient.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err = helpers.CreateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}

	if err = helpers.UpdateConfigTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// ManagementIngressOperatorCatalogUpdate is for testing the update of the ManagementIngressOperatorCatalog
func ManagementIngressOperatorCatalogUpdate(t *testing.T) {

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global
	olmClient, err := olmclient.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err = helpers.CreateTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}

	if err = helpers.UpdateCatalogTest(olmClient, f, ctx); err != nil {
		t.Fatal(err)
	}
}
