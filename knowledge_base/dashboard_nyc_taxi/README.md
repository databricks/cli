# Dashboard for NYC Taxi Trip Analysis

This example shows how to define a Databricks Asset Bundle with an AI/BI dashboard and a job that captures a snapshot of the dashboard and emails it to a subscriber.

It deploys the sample __NYC Taxi Trip Analysis__ dashboard to a Databricks workspace and configures a daily schedule to run the dashboard and send the snapshot in email to a specified email address.

For more information about AI/BI dashboards, please refer to the [documentation](https://docs.databricks.com/dashboards/index.html).

## Prerequisites

This example includes a dashboard snapshot task, which requires Databricks CLI  v0.250.0 or above. Creating dashboards in bundles is supported in Databricks CLI v0.232.0 or above.

## Usage

1. Modify `databricks.yml`:
    - Update the `host` field under `workspace` to the Databricks workspace to deploy to.
    - Update the `warehouse` field under `warehouse_id` to the name of the SQL warehouse to use.

2. Modify `resources/nyc_taxi_trip_analysis.job.yml`:
    - Update the `user_name` field under `subscribers` to the dashboard subscriber's email.

3. Deploy the dashboard:
    - Run `databricks bundle deploy` to deploy the dashboard.
    - Run `databricks bundle open` to navigate to the deployed dashboard in your browser. Alternatively, run `databricks bundle summary` to display its URL.

The AI/BI dashboard is created and the snapshot job is set to run daily at 8 AM, which captures a snapshot of the dashboard, and sends it in email to the specified subscriber.

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
