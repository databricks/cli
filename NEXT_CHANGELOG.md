# NEXT CHANGELOG

## Release v1.0.0

### Notable Changes

* The Databricks CLI is now generally available with version v1.0.0 as the first major release 🚀. From this version on, the CLI follows semantic versioning (see [README](README.md)). This change does not impact DABs or other existing commands beyond the changes listed below.
* The 0.299.x line continues to receive security-critical patches through May 20, 2027; see [SECURITY](SECURITY.md) for the support policy.
* Starting with v1.0.0, the CLI will use immutable release tags, providing users with better security guarantees.

### CLI

* Added `databricks aitools` command group for installing Databricks skills into your coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity). Skills are fetched from [github.com/databricks/databricks-agent-skills](https://github.com/databricks/databricks-agent-skills) and either symlinked into each agent's skills directory or copied into the current project. Use `databricks aitools install` to set up, `update` to pull newer versions, `list` to see what's available, and `uninstall` to remove them.
* `[__settings__].default_profile` is now consulted as a fallback by `databricks api`, `databricks auth token`, and bundle commands when neither `--profile` nor `DATABRICKS_CONFIG_PROFILE` is set. `databricks auth token` continues to give precedence to `DATABRICKS_HOST` over `default_profile`. For bundle commands, `default_profile` only applies when the bundle does not pin its own `workspace.host`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.
