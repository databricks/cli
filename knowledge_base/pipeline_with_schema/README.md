# Pipeline with a dedicated Unity Catalog schema

This example demonstrates how to define a Unity Catalog schema and a Delta Live Tables pipeline that uses it.

## Prerequisites

* Databricks CLI v0.225.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Update the `catalog_name` field in `resources/schema.yml` to a Unity Catalog catalog where you are granted
permission to create schemas. This can be `main`, `sandbox`, or any other catalog you have access to.

Run `databricks bundle deploy` to create the schema and deploy the pipeline.

Run `databricks bundle run example_pipeline` to run a pipeline update.

Example output:

```
% databricks bundle run example_pipeline
Update URL: https://...

2024-08-15T09:51:02.163Z update_progress INFO "Update 096257 is WAITING_FOR_RESOURCES."
2024-08-15T09:55:24.821Z update_progress INFO "Update 096257 is INITIALIZING."
2024-08-15T09:55:27.097Z update_progress INFO "Update 096257 is SETTING_UP_TABLES."
2024-08-15T09:55:33.466Z update_progress INFO "Update 096257 is RUNNING."
2024-08-15T09:55:33.479Z flow_progress   INFO "Flow 'range' is QUEUED."
2024-08-15T09:55:33.504Z flow_progress   INFO "Flow 'double' is QUEUED."
2024-08-15T09:55:33.518Z flow_progress   INFO "Flow 'range' is PLANNING."
2024-08-15T09:55:34.060Z flow_progress   INFO "Flow 'range' is STARTING."
2024-08-15T09:55:34.118Z flow_progress   INFO "Flow 'range' is RUNNING."
2024-08-15T09:55:41.319Z flow_progress   INFO "Flow 'range' has COMPLETED."
2024-08-15T09:55:42.205Z flow_progress   INFO "Flow 'double' is PLANNING."
2024-08-15T09:55:43.492Z flow_progress   INFO "Flow 'double' is STARTING."
2024-08-15T09:55:43.513Z flow_progress   INFO "Flow 'double' is RUNNING."
2024-08-15T09:55:47.550Z flow_progress   INFO "Flow 'double' has COMPLETED."
2024-08-15T09:55:48.594Z update_progress INFO "Update 096257 is COMPLETED."
```
