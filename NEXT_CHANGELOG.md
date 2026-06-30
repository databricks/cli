# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

 * `databricks experimental genie ask` now supports multi-turn conversations via `-s`/`--session`: reuse a session label (any string you pick) to continue a conversation across calls. The label maps to the server conversation id in `~/.databricks/genie-conversations.json`, and an expired conversation transparently starts fresh. Also cancels cleanly on SIGINT and SIGTERM ([#5762](https://github.com/databricks/cli/pull/5762)).
 * `databricks aitools install` is now plugin-first: it installs the Databricks plugin through each agent's own CLI (Claude Code, Codex, GitHub Copilot) instead of copying raw skill files. Agents without a plugin (OpenCode, Antigravity) still get skill files, and Cursor prints the `/add-plugin databricks` step. Use `--skills-only` to force raw skill files for every agent, or `--path <dir>` to write skills to a directory ([#5738](https://github.com/databricks/cli/pull/5738)).

### Bundles

 * direct: Fixed persistent drift on `model_serving_endpoints` caused by the `traffic_config` field ([#5708](https://github.com/databricks/cli/pull/5708)).
 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).

### Dependency updates

### API Changes
