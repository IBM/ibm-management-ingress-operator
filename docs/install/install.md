**Table of Contents**

- [Install the management ingress operator](#install-the-ibm-management-ingress-operator)
    - [Install the management ingress operator On OCP 4.x](#install-the-ibm-management-ingress-operator-on-ocp-4x)
        - [1. Create OperatorSource](#1-create-operatorsource)
        - [2. Create a Namespace `ibm-management-ingress-operator`](#2-create-a-namespace-ibm-management-ingress-operator)
        - [3. Install meta Operator](#3-install-ibm-management-ingress-operator)
        - [4. Check the installed operators](#4-check-the-installed-operators)
    - [Post-installation](#post-installation)

# Install the management ingress operator

## Install the management ingress operator On OCP 4.x

### 1. Create OperatorSource

Open OCP console, click the `Plus` button on the top right and paste the following content, then click `Create`.

```yaml
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: opencloud-operators
  namespace: openshift-marketplace
spec:
  authorizationToken: {}
  displayName: IBMCS Operators
  endpoint: https://quay.io/cnr
  publisher: IBM
  registryNamespace: opencloudio
  type: appregistry
```

### 2. Create a Namespace `ibm-management-ingress-operator`

Open the `OperatorHub` page in OCP console left menu, then `Create Project`, e.g., create a project named `ibm-management-ingress-operator`.

### 3. Install meta Operator

Open `OperatorHub` and search `ibm-management-ingress-operator` to find the operator, and install it.

### 4. Check the installed operators

Open `Installed Operators` page to check the installed operators.

## Post-installation

The management ingress operators and their custom resource would be deployed in the cluster.
