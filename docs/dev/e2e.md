**Table of Contents**

- [Running E2E Tests](#running-e2e-tests)
    - [Reference](#reference)

# Running E2E Tests

1. Ensure **operator-sdk** is installed and login to your OpenShift cluster as an admin user.

1. If the namespace **ibm-management-ingress-operator** exists, delete it. Do this to make sure the namespace is clean.

    ```bash
    oc delete namespace ibm-management-ingress-operator
    ```

1. Create the namespace **ibm-management-ingress-operator**.

    ```bash
    oc create namespace ibm-management-ingress-operator
    ```

1. Run the test using `make test-e2e`  command locally.

    ```bash
    make test-e2e
    ```

## Reference

- [Running tests](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-the-tests)
- [Installing Operator-SDK](https://github.com/operator-framework/operator-sdk#quick-start)
