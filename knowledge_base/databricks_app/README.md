# Databricks App for working with Databricks jobs

This example demonstrates how to define an Databricks App in a Databricks Asset Bundle.

It includes and deploys an example app and a job managed by DABs to a Databricks workspace.
The app shows current status of the job and lists all existing runs.

For more information about Databricks Apps, please refer to the [documentation](https://docs.databricks.com/en/dev-tools/databricks-apps/index.html).

## Prerequisites

* Databricks CLI v0.238.0 or above

## Usage

Modify `databricks.yml`:
* Update the `host` field under `workspace` to the Databricks workspace to deploy to.

Run `databricks bundle deploy` to deploy the app.

Run `databricks bundle run job_manager` to start the app and execute app deployment. 
When you change app code or config, you need to run `databricks bundle deploy` and `databricks bundle run job_manager` to update the app source code and configuration.

Run `databricks bundle open` to navigate to the deployed app in your browser. Alternatively, run `databricks bundle summary` to display its URL.