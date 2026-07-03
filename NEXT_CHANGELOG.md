# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).

### Bundles
 * direct: add basic version of job_runs resource (experimental) ([#5603](https://github.com/databricks/cli/pull/5603)).
 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
 * `bundle generate job` now downloads workspace files referenced by `spark_python_task`, rewriting them to a relative path like it already does for notebooks. Git-sourced files and cloud URIs are left untouched ([#5799](https://github.com/databricks/cli/pull/5799)).

### Dependency updates

### API Changes
