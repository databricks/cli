# Development cluster

This example demonstrates how to define and use a development (all-purpose) cluster in a Databricks Asset Bundle.

This bundle defines an `example_job` which is run on a job cluster in production mode.

For the development mode (default `dev` target) the job is overriden to use a development cluster which is provisioned 
as part of the bundle deployment as well.

For more information, please refer to the [documentation](https://docs.databricks.com/en/dev-tools/bundles/settings.html#clusters).

## Prerequisites

* Databricks CLI v0.229.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the job. It's deployed to `dev` target with a defined `development_cluster` cluster.

Run `databricks bundle deploy -t prod` to deploy the job to prod target. It's deployed with a job cluster instead of development one.

Run `databricks bundle run example_job` to run the job.
