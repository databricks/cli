# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).
* An explicit `--profile` now takes precedence over authentication environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.). Previously these env vars silently shadowed the selected profile's host and credentials ([#5096](https://github.com/databricks/cli/issues/5096)).

### Bundles

 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).

### Dependency updates

### API Changes
