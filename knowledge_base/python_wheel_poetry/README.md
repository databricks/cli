# Python wheel with Poetry

This example demonstrates how to use Poetry with a Databricks Asset Bundle.

## Prerequisites

* Databricks CLI v0.209.0 or above
* Python 3.10 or above
* Poetry 1.6 or above (install with `pip3 install poetry`)

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Optional: update the `node_type_id` to the node type relevant to your cloud in `resources/job.yml`.

Run `databricks bundle deploy` to build the wheel, upload the wheel, and deploy the job.

Run `databricks bundle run example_job` to run the job that calls into the wheel.
