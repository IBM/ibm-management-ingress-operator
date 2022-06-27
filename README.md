# ibm-management-ingress-operator

> **Important:** Do not install this operator directly. Only install this operator using the IBM Common Services Operator. For more information about installing this operator and other Common Services operators, see [Installer documentation](http://ibm.biz/cpcs_opinstall). If you are using this operator as part of an IBM Cloud Pak, see the documentation for that IBM Cloud Pak to learn more about how to install and use the operator service. For more information about IBM Cloud Paks, see [IBM Cloud Paks that use Common Services](http://ibm.biz/cpcs_cloudpaks).

You can use the ibm-management-ingress-operator to install the IBM System Management Ingress service, which includes a Nginx-based ingress controller. You can use the IBM System Management Ingress service to expose your services when you install and use an IBM Cloud Pak or IBM Cloud Platform Common Services.

For more information about the available IBM Cloud Platform Common Services, see the [IBM Knowledge Center](http://ibm.biz/cpcsdocs).

## Supported platforms

Red Hat OpenShift Container Platform 4.3 or newer installed on one of the following platforms:

- Linux x86_64
- Linux on Power (ppc64le)
- Linux on IBM Z and LinuxONE

## Operator versions

- 1.1.0
- 1.2.1
    - Support for OpenShift 4.3 and 4.4.
- 1.3.3
- 1.4.2
- 1.5.1
- 1.6.0
- 1.7.0
- 1.7.1
- 1.8.0
- 1.9.0
- 1.11.0
- 1.10.0
- 1.12.0
- 1.13.0
- 1.14.0
- 1.16.0
- 1.17.0

## Prerequisites

Before you install this operator, you need to first install the operator dependencies and prerequisites:

- For the list of operator dependencies, see the IBM Knowledge Center [Common Services dependencies documentation](http://ibm.biz/cpcs_opdependencies).
- For the list of prerequisites for installing the operator, see the IBM Knowledge Center [Preparing to install services documentation](http://ibm.biz/cpcs_opinstprereq).

## Documentation

To install the operator with the IBM Common Services Operator follow the the installation and configuration instructions within the IBM Knowledge Center.

- If you are using the operator as part of an IBM Cloud Pak, see the documentation for that IBM Cloud Pak. For a list of IBM Cloud Paks, see [IBM Cloud Paks that use Common Services](http://ibm.biz/cpcs_cloudpaks).
- If you are using the operator with an IBM Containerized Software, see the IBM Cloud Platform Common Services Knowledge Center [Installer documentation](http://ibm.biz/cpcs_opinstall).

## SecurityContextConstraints Requirements

The IBM System Management ingress service supports running with the OpenShift Container Platform default restricted Security Context Constraints (SCCs).

For more information about the OpenShift Container Platform Security Context Constraints, see [Managing Security Context Constraints](https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html).

## Developer guide

If, as a developer, you are looking to build and test this operator to try out and learn more about the operator and its capabilities, you can use the following developer guide. This guide provides commands for a quick install and initial validation for running the operator.

> **Important:** The following developer guide is provided as-is and only for trial and education purposes. IBM and IBM Support does not provide any support for the usage of the operator with this developer guide. For the official supported install and usage guide for the operator, see the the IBM Knowledge Center documentation for your IBM Cloud Pak or for IBM Cloud Platform Common Services.

### Quick start guide

Use the following quick start commands for building and testing the operator:

#### Cloning the repository

Check out the ibm-management-ingress-operator repository.

```bash
# git clone https://github.com/IBM/ibm-management-ingress-operator.git
# cd ibm-management-ingress-operator
```

#### Building the operator

Build the ibm-management-ingress-operator image and push it to a public registry, such as Quay.io.

```bash
# make images
```

#### Using the image

Edit `deploy/operator.yaml` and replace the image name.

```bash
vim deploy/operator.yaml
```

#### Installing

```bash
# kubectl apply -f deploy/
deployment.apps/ibm-management-ingress-operator created
role.rbac.authorization.k8s.io/ibm-management-ingress-operator created
clusterrole.rbac.authorization.k8s.io/ibm-management-ingress-operator created
rolebinding.rbac.authorization.k8s.io/ibm-management-ingress-operator created
clusterrolebinding.rbac.authorization.k8s.io/ibm-management-ingress-operator created
serviceaccount/ibm-management-ingress-operator created
```

```bash
# kubectl get pods
NAME                                               READY   STATUS    RESTARTS   AGE
ibm-management-ingress-operator-686fdb84f8-cxqc7   1/1     Running   0          62s
management-ingress-5b5b66dcd7-hfnpm                1/1     Running   0          33s
```

#### Uninstalling

```bash
# kubectl delete -f deploy/
```

### Debugging guide

Use the following commands to debug the operator:

```bash
# kubectl logs deployment.apps/ibm-management-ingress-operator -n <namespace>
```

### End-to-End testing

For more instructions on how to run end-to-end testing with the Operand Deployment Lifecycle Manager, see [ODLM guide](https://github.com/IBM/operand-deployment-lifecycle-manager/blob/master/docs/install/common-service-integration.md#end-to-end-test).
