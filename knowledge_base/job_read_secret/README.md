# Databricks job that reads a secret from a secret scope

This example demonstrates how to define a secret scope and a job with a task that reads from it in a Databricks Asset Bundle.

It includes and deploys an example secret scope, and a job with a task in a bundle that reads a secret from the secret scope to a Databricks workspace.

For more information about Databricks secrets, see the [documentation](https://docs.databricks.com/aws/en/security/secrets).

## Prerequisites

* Databricks CLI v0.252.0 or above

## Usage

Modify `databricks.yml`:
* Update the `host` field under `workspace` to the Databricks workspace to deploy to

Run `databricks bundle deploy` to deploy the bundle.

Run this script to write a secret to the secret scope. Databricks CLI commands run from inside the bundle root directory use the same authentication credentials as the bundle:

```
SECRET_SCOPE_NAME=$(databricks bundle summary -o json | jq -r '.resources.secret_scopes.my_secret_scope.name')

databricks secrets put-secret ${SECRET_SCOPE_NAME} example-key --string-value example-value
```

Run the job:
```
databricks bundle run example_python_job
```