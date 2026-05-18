# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* Added `databricks aitools` command group for installing Databricks skills into your coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity). Skills are fetched from [github.com/databricks/databricks-agent-skills](https://github.com/databricks/databricks-agent-skills) and either symlinked into each agent's skills directory or copied into the current project. Use `databricks aitools install` to set up, `update` to pull newer versions, `list` to see what's available, and `uninstall` to remove them.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))
* Add `postgres_catalogs` resource to bind a Unity Catalog catalog to a Postgres database on a Lakebase Autoscaling branch.
* Add `postgres_synced_tables` resource to manage Lakebase synced tables that replicate Unity Catalog Delta tables into Postgres. Direct deployment engine only.

### Dependency updates
