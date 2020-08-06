#!/bin/bash
#
# Copyright 2020 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL:=help
# Specify whether this repo is build locally or not, default values is '1';
# If set to 1, then you need to also set 'DOCKER_USERNAME' and 'DOCKER_PASSWORD'
# environment variables before build the repo.
BUILD_LOCALLY ?= 1
TARGET_GOOS=linux
TARGET_GOARCH=amd64

# The namespce that operator will be deployed in
NAMESPACE=ibm-management-ingress-operator

# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the IMG and REGISTRY environment variable.
IMG ?= ibm-management-ingress-operator
REGISTRY ?= "hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom"
CSV_VERSION ?= 1.3.0

IMAGE_REPO ?= "hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom"
IMAGE_NAME ?= ibm-management-ingress-operator

QUAY_USERNAME ?=
QUAY_PASSWORD ?=

MARKDOWN_LINT_WHITELIST=https://quay.io/cnr

TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
#VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')
LOCAL_OS := $(shell uname)
LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
    STRIP_FLAGS=
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
    STRIP_FLAGS="-x"
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif

include common/Makefile.common.mk

##@ Application

install: ## Install all resources (CR/CRD's, RBCA and Operator)
	@echo ....... Set environment variables ......
	- export WATCH_NAMESPACE=${NAMESPACE}
	@echo ....... Creating CRDS .......
	- kubectl apply -f deploy/crds/operator.ibm.com_managementingresses_crd.yaml
	@echo ....... Creating RBAC .......
	- kubectl apply -f deploy/service_account.yaml -n ${NAMESPACE}
	- kubectl apply -f deploy/role.yaml
	- kubectl apply -f deploy/role_binding.yaml
	@echo ....... Creating Operator .......
	- kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}
	@echo ....... Creating CR .......
	- kubectl apply -f deploy/crds/operator.ibm.com_v1alpha1_managementingress_cr.yaml -n ${NAMESPACE}

uninstall: ## Delete all that performed in $ make install
	@echo ....... Deleting CR .......
	- kubectl delete -f deploy/crds/operator.ibm.com_v1alpha1_managementingress_cr.yaml -n ${NAMESPACE}
	@echo ....... Deleting Operator .......
	- kubectl delete -f deploy/operator.yaml -n ${NAMESPACE}
	@echo ....... Deleting CRDs .......
	- kubectl delete -f deploy/crds/operator.ibm.com_managementingresses_crd.yaml
	@echo ....... Deleting RBAC and Service Account .......
	- kubectl delete -f deploy/role_binding.yaml
	- kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	- kubectl delete -f deploy/role.yaml

##@ Development

check: lint-all ## Check all files lint error

code-dev: ## Run the default dev commands which are go tidy, fmt, vet then execute the $ make code-gen
	@echo Running the common required commands for developments purposes
	- make code-tidy
	- make code-fmt
	- make code-vet
	- make code-gen
	@echo Running the common required commands for code delivery
	- make check
	- make test
	- make build

run: ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./cmd/manager/main.go

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
endif

##@ Build

build:
	@echo "Building ibm-management-ingress-operator binary"
	@CGO_ENABLED=0 go build -o build/_output/bin/$(IMG) ./cmd/manager
	@strip $(STRIP_FLAGS) build/_output/bin/$(IMG)

build-image-amd64: build $(CONFIG_DOCKER_TARGET)
	@echo "Building ibm-management-ingress-operator amd64 image"
	$(eval ARCH := $(shell uname -m|sed 's/x86_64/amd64/'))
	docker build -t $(REGISTRY)/$(IMG)-$(ARCH):$(VERSION) -f build/Dockerfile .
	@\rm -f build/_output/bin/ibm-management-ingress-operator
	@if [ $(BUILD_LOCALLY) -ne 1 ] && [ "$(ARCH)" = "amd64" ]; then docker push $(REGISTRY)/$(IMG)-$(ARCH):$(VERSION); fi

push-image-amd64: build-image-amd64
	@docker push $(REGISTRY)/$(IMG)-amd64:$(VERSION)

# runs on amd64 machine
build-image-ppc64le: $(CONFIG_DOCKER_TARGET)
ifeq ($(LOCAL_OS),Linux)
ifeq ($(LOCAL_ARCH),x86_64)
	@echo "Building ibm-management-ingress-operator ppc64le image"
	GOOS=linux GOARCH=ppc64le CGO_ENABLED=0 go build -o build/_output/bin/ibm-management-ingress-operator-ppc64le ./cmd/manager
	docker run --rm --privileged multiarch/qemu-user-static:register --reset
	docker build -t $(REGISTRY)/$(IMG)-ppc64le:$(VERSION) -f build/Dockerfile.ppc64le .
	@\rm -f build/_output/bin/ibm-management-ingress-operator-ppc64le
	@if [ $(BUILD_LOCALLY) -ne 1 ]; then docker push $(REGISTRY)/$(IMG)-ppc64le:$(VERSION); fi
endif
endif

push-image-ppc64le: build-image-ppc64le
ifeq ($(LOCAL_OS),Linux)
ifeq ($(LOCAL_ARCH),x86_64)
	@docker push $(REGISTRY)/$(IMG)-ppc64le:$(VERSION)
endif
endif

# runs on amd64 machine
build-image-s390x: $(CONFIG_DOCKER_TARGET)
ifeq ($(LOCAL_OS),Linux)
ifeq ($(LOCAL_ARCH),x86_64)
	@echo "Building ibm-management-ingress-operator s390x image"
	GOOS=linux GOARCH=s390x CGO_ENABLED=0 go build -o build/_output/bin/ibm-management-ingress-operator-s390x ./cmd/manager
	docker run --rm --privileged multiarch/qemu-user-static:register --reset
	docker build -t $(REGISTRY)/$(IMG)-s390x:$(VERSION) -f build/Dockerfile.s390x .
	@\rm -f build/_output/bin/ibm-management-ingress-operator-s390x
	@if [ $(BUILD_LOCALLY) -ne 1 ]; then docker push $(REGISTRY)/$(IMG)-s390x:$(VERSION); fi
endif
endif

push-image-s390x: build-image-s390x
ifeq ($(LOCAL_OS),Linux)
ifeq ($(LOCAL_ARCH),x86_64)
	@docker push $(REGISTRY)/$(IMG)-s390x:$(VERSION)
endif
endif

##@ Test

test: ## Run unit test
	@go test ${TESTARGS} ./pkg/...

test-e2e: ## Run integration e2e tests with different options.
	@echo ... Running locally ...
	- operator-sdk test local ./test/e2e --verbose --up-local --namespace=${NAMESPACE}

coverage: ## Run code coverage test
	@common/scripts/codecov.sh ${BUILD_LOCALLY}

scorecard: ## Run scorecard test
	@echo ... Running the scorecard test
	- operator-sdk scorecard --verbose

##@ Release

build-images: build-image-amd64 build-image-ppc64le build-image-s390x

push-images: push-image-amd64 push-image-ppc64le push-image-s390x

build-push-image: build-images push-images multiarch-image

# multiarch-image section
multiarch-image: $(CONFIG_DOCKER_TARGET)
	@MAX_PULLING_RETRY=20 RETRY_INTERVAL=30 common/scripts/multiarch_image.sh $(IMAGE_REPO) $(IMAGE_NAME) $(VERSION)

images: build-push-image

csv: ## Push CSV package to the catalog
	@RELEASE=${CSV_VERSION} common/scripts/push-csv.sh

all: check test coverage build images

##@ Cleanup
clean: ## Clean build binary
	rm -f build/_output/bin/$(IMG)

##@ Help
help: ## Display this help
	@echo "Usage:\n  make \033[36m<target>\033[0m"
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: all build run check install uninstall code-dev test test-e2e coverage images csv clean help multiarch-image
