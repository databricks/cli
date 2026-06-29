# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
* Allow bundles with `apps` resources to have a top-level `run_as` identity configured. Apps do not support `run_as` via the API and are simply skipped; other resources (jobs, pipelines, etc.) continue to have `run_as` applied as before.

### Dependency updates

### API Changes
