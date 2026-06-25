# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Fix spurious update when `apply_policy_default_values: true` is set on job task, for-each-task, or job cluster new_cluster ([#5731](https://github.com/databricks/cli/pull/5731)). Also fix spurious updates for for-each-task clusters due to missing backend defaults for `data_security_mode`, `node_type_id`, `driver_node_type_id`, `driver_instance_pool_id`, `enable_elastic_disk`, and `enable_local_disk_encryption`.
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
 * Expose a computed, read-only `volume_path` on `resources.volumes.*` so configs can reference a volume's Unity Catalog path via `${resources.volumes.<key>.volume_path}` instead of hardcoding `/Volumes/<catalog>/<schema>/<name>` ([#5550](https://github.com/databricks/cli/pull/5550)).
   * `volume_path` is derived purely from the volume's `catalog_name`, `schema_name`, and `name`, so the reference is resolved early (at initialize) and inlined into the referring field. Referencing `volume_path` therefore does not make the referring resource depend on the volume during deploy; if `catalog_name`/`schema_name`/`name` themselves reference other resources, the referrer depends on those resources instead.
   * Supported on the direct deployment engine (`DATABRICKS_BUNDLE_ENGINE=direct`). On the Terraform engine `volume_path` is dropped before apply, and a `volume_path` whose components embed a value only known after deploy (for example `${resources.jobs.<key>.creator_user_name}`) is not supported.

### Dependency updates

### API Changes
