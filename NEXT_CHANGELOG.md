# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).
 * `databricks labs list` now only shows projects that can be installed ([#5560](https://github.com/databricks/cli/pull/5560)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Fix spurious update when `apply_policy_default_values: true` is set on job task, for-each-task, or job cluster new_cluster ([#5731](https://github.com/databricks/cli/pull/5731)). Also fix spurious updates for for-each-task clusters due to missing backend defaults for `data_security_mode`, `node_type_id`, `driver_node_type_id`, `driver_instance_pool_id`, `enable_elastic_disk`, and `enable_local_disk_encryption`.
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
 * Allow bundles with `apps` resources to have a top-level `run_as` identity configured. Apps do not support `run_as` via the API and are simply skipped; other resources (jobs, pipelines, etc.) continue to have `run_as` applied as before.
 * Fixed `bundle deployment migrate` failing on `model_serving_endpoints`/`database_instances` with permissions (regression since v1.5.0) ([#5775](https://github.com/databricks/cli/pull/5775)).

### Dependency updates

### API Changes
