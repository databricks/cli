The types in this package are equivalent to the lumberjack protos defined in Universe.
You can find all lumberjack protos for the Databricks CLI in the `proto/logs/frontend/databricks_cli` directory.

`databricks air run` logs one `air_run_event` per submission: requested compute,
which config options were set, the outcome and latency, and how long packaging and
uploading the code snapshot took. It never includes code, paths, names, parameters,
or image references. Set `DATABRICKS_CLI_DISABLE_TELEMETRY=1` to opt out.
