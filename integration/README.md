# Integration tests

This directory contains integration tests for the project.

## Deprecated: do not add tests here

This tree is deprecated and should not be extended with new tests. A test here only ever runs against a real workspace, so it contributes nothing to the local suite and is only exercised when someone runs a cloud job.

Write an acceptance test instead (see `acceptance/README.md`) and set `Cloud = true` in its `test.toml`. One test then covers both: it runs locally against the fake server in `libs/testserver` on every `./task test`, *and* against a real workspace when `CLOUD_ENV` is set. Existing tests here are still maintained; extend them only when there is no acceptance-test equivalent.

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
./task integration
```
