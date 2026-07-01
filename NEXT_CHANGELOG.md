# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `ssh connect` now accepts a `--base-environment` flag to run a serverless session on a custom base environment. It takes an `env.yaml` path, a `workspace-base-environments/...` resource ID, or a base environment display name, and is rejected together with `--environment-version` or `--cluster` ([#5706](https://github.com/databricks/cli/pull/5706)).
 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).
 * `databricks labs list` now only shows projects that can be installed ([#5560](https://github.com/databricks/cli/pull/5560)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Fix spurious update when `apply_policy_default_values: true` is set on job task, for-each-task, or job cluster new_cluster ([#5731](https://github.com/databricks/cli/pull/5731)). Also fix spurious updates for for-each-task clusters due to missing backend defaults for `data_security_mode`, `node_type_id`, `driver_node_type_id`, `driver_instance_pool_id`, `enable_elastic_disk`, and `enable_local_disk_encryption`.
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
 * Fixed `bundle deployment migrate` failing on `model_serving_endpoints`/`database_instances` with permissions (regression since v1.5.0) ([#5775](https://github.com/databricks/cli/pull/5775)).
 * After a terraform deploy, the CLI now dry-runs an in-memory migration to the direct engine (writing nothing locally or remotely) and reports the outcome via telemetry, warning if the migration could not be completed.

### Dependency updates

### API Changes
