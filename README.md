<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Management Ingress Operator](#ibm-management-ingress-operator)
    - [Overview](#overview)
    - [Prerequisites](#prerequisites)
        - [PodSecurityPolicy Requirements](#podsecuritypolicy-requirements)
        - [PodSecurityPolicy Requirements](#securitycontextconstraints-requirements)
    - [Getting Started](#getting-started)
        - [Cloning the repository](#cloning-the-repository)
        - [Building the operator](#building-the-operator)
        - [Installing](#installing)
        - [Uninstalling](#uninstalling)
        - [Troubleshooting](#troubleshooting)
        - [Running Tests](#running-tests)
        - [Development](#development)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Management Ingress Operator

## Overview

This is the operator to deploy management ingress of IBM Cloud Pak foundation..

## Prerequisites

- [go][go_tool] version v1.13+.
- [docker][docker_tool] version 17.03+
- [kubectl][kubectl_tool] v1.13.0+
- [operator-sdk][operator_install]
- Access to a Kubernetes v1.13.0+ cluster

### PodSecurityPolicy Requirements

### SecurityContextConstraints Requirements

The management service supports running under the OpenShift Container Platform default restricted security context constraints.

## Getting Started

### Cloning the repository

Checkout this Management Ingress Operator repository

```bash
# git clone https://github.com/IBM/ibm-management-ingress-operator.git
# cd ibm-management-ingress-operator
```

### Building the operator

Build the meta operator image and push it to a public registry, such as quay.io:

```bash
# make build
# make images
```

### Installing

Run `make install` to install the operator. Check that the operator is running in the cluster.

Following the expected result.

```bash
# kubectl get all -n ibm-management-ingress-operator
NAME                                           READY   STATUS    RESTARTS   AGE
pod/ibm-management-ingress-operator-786d699956-z7k4n   1/1     Running   0          21s

NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/ibm-management-ingress-operator   1/1     1            1           22s

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/ibm-management-ingress-operator-786d699956   1         1         1       22s
```

### Uninstalling

To uninstall all that was performed in the above step run `make uninstall`.

### Troubleshooting

Use the following command to check the operator logs.

```bash
# kubectl logs deployment.apps/ibm-management-ingress-operator -n ibm-management-ingress-operator
```

### Running Tests

[End to end tests](./docs/e2e.md)
For more information see the [writing e2e tests](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md) guide.

### Development

When the API or CRD changed, run `make code-dev` re-generate the code.

[go_tool]: https://golang.org/dl/
[kubectl_tool]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[docker_tool]: https://docs.docker.com/install/
[operator_sdk]: https://github.com/operator-framework/operator-sdk
[operator_install]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user/install-operator-sdk.md
