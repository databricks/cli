# Spark JAR task example

This example demonstrates how to define and use a Spark JAR Task.

## Prerequisites

* Databricks CLI v0.224.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Update the `artifact_path` field under `workspace` in `databricks.yml` to the Databricks Volume path where the JAR artifact needs to be deployed.

Run `databricks bundle deploy` to deploy the job.

Run `databricks bundle run spark_jar_job` to run the job.

Example output:

```
$ databricks bundle run spark_jar_job
Run URL: https://...


```
