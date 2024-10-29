# Dashboard for NYC Taxi Trip Analysis

This example demonstrates how to define an AI/BI dashboard in a Databricks Asset Bundle.

It includes and deploys the sample __NYC Taxi Trip Analysis__ dashboard to a Databricks workspace.

For more information about AI/BI dashboards, please refer to the [documentation](https://docs.databricks.com/dashboards/index.html).

## Prerequisites

* Databricks CLI v0.232.0 or above

## Usage

Modify `databricks.yml`:
* Update the `host` field under `workspace` to the Databricks workspace to deploy to.
* Update the `warehouse` field under `warehouse_id` to the name of the SQL warehouse to use.

Run `databricks bundle deploy` to deploy the dashboard.

Run `databricks bundle open` to navigate to the deployed dashboard in your browser. Alternatively, run `databricks bundle summary` to display its URL.

### Visual modification

You can use the Databricks UI to modify the dashboard, but any modifications made through the UI will not be applied to the bundle `.lvdash.json` file unless you explicitly update it. 

To update the local bundle `.lvdash.json` file, run:

```sh
databricks bundle generate dashboard --resource nyc_taxi_trip_analysis --force
```

To continuously poll and retrieve the updated `.lvdash.json` file when it changes, run:

```sh
databricks bundle generate dashboard --resource nyc_taxi_trip_analysis --force --watch
```

Any remote modifications of a dashboard are noticed by the `deploy` command and require
you to acknowledge that remote changes can be overwritten by local changes.
It is therefore recommended to run the `generate` command before running the `deploy` command.
Otherwise, you may lose your remote changes.

### Manual modification

You can modify the `.lvdash.json` file directly and redeploy to observe your changes.
