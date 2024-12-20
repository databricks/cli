# Save job result to volume

This example demonstrates how to define and use a Unity Catalog Volume in a Databricks Asset Bundle.

Specifically we'll define a `hello_world_job` job which writes "Hello, World!"
to a file in a Unity Catalog Volume.

The bundle also defines a Volume and the associated Schema in which the Job writes text to.

## Prerequisites

* Databricks CLI v0.236.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the job.

Run `databricks bundle run hello_world_job` to run the job and store the results in UC volume.
