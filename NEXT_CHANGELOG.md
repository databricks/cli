# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* experimental `ssh connect`: bare `python`/`pip` in an interactive session now resolve to the environment interpreter (`$DATABRICKS_VIRTUAL_ENV`) instead of the system or cluster-libraries interpreter, so packages installed in the environment are importable without extra setup. The interactive shell is now non-login (`bash -i`) and the server seeds a `~/.bashrc` snippet that re-prepends the environment's bin directory to `PATH` ([#5888](https://github.com/databricks/cli/pull/5888)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Fix spurious update when `apply_policy_default_values: true` is set on job task, for-each-task, or job cluster new_cluster ([#5731](https://github.com/databricks/cli/pull/5731)). Also fix spurious updates for for-each-task clusters due to missing backend defaults for `data_security_mode`, `node_type_id`, `driver_node_type_id`, `driver_instance_pool_id`, `enable_elastic_disk`, and `enable_local_disk_encryption`.
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
 * Allow bundles with `apps` resources to have a top-level `run_as` identity configured. Apps do not support `run_as` via the API and are simply skipped; other resources (jobs, pipelines, etc.) continue to have `run_as` applied as before.
 * Fixed `bundle deployment migrate` failing on `model_serving_endpoints`/`database_instances` with permissions (regression since v1.5.0) ([#5775](https://github.com/databricks/cli/pull/5775)).
 * Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
 * Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
 * Added an `env:` section to `scripts.<name>` for declaring environment variables that may reference `${bundle.*}`, `${workspace.*}`, and `${var.*}` ([#4179](https://github.com/databricks/cli/issues/4179)).

### Dependency updates

### API Changes
