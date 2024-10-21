# Job with `run_job` tasks

This example demonstrates how to compose multiple jobs with `run_job` tasks.

## Prerequisites

* Databricks CLI v0.230.0 or above
* Serverless enabled on the Databricks workspace

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the bundle.

Run `databricks bundle run primary` to run the primary job that triggers the leaf jobs.
