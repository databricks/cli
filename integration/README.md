# Integration tests

This directory contains integration tests for the project.

The tree structure generally mirrors the source code tree structure.

Requirements for new files in this directory:
* Every package **must** be named after its directory with `_test` appended
  * Requiring a different package name for integration tests avoids aliasing with the main package.
* Every integration test package **must** include a `main_test.go` file.

These requirements are enforced by a unit test in this directory.

## Running integration tests

Integration tests require the following environment variables:
* `CLOUD_ENV` - set to the cloud environment to use (e.g. `aws`, `azure`, `gcp`)
* `DATABRICKS_HOST` - set to the Databricks workspace to use
* `DATABRICKS_TOKEN` - set to the Databricks token to use

Optional environment variables:
* `TEST_DEFAULT_WAREHOUSE_ID` - set to the default warehouse ID to use
* `TEST_METASTORE_ID` - set to the metastore ID to use
* `TEST_INSTANCE_POOL_ID` - set to the instance pool ID to use
* `TEST_BRICKS_CLUSTER_ID` - set to the cluster ID to use

To run all integration tests, use the following command:

```bash
go test ./integration/...
```

Alternatively:

```bash
make integration
```
