# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* Added `databricks aitools` command group for installing Databricks skills into your coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity). Skills are fetched from [github.com/databricks/databricks-agent-skills](https://github.com/databricks/databricks-agent-skills) and either symlinked into each agent's skills directory or copied into the current project. Use `databricks aitools install` to set up, `update` to pull newer versions, `list` to see what's available, and `uninstall` to remove them.
* `[__settings__].default_profile` is now consulted as a fallback by `databricks api`, `databricks auth token`, and bundle commands when neither `--profile` nor `DATABRICKS_CONFIG_PROFILE` is set. `databricks auth token` continues to give precedence to `DATABRICKS_HOST` over `default_profile`. For bundle commands, `default_profile` only applies when the bundle does not pin its own `workspace.host`.
* Extend the default-profile fallback to also pin the `[DEFAULT]` section when `[__settings__].default_profile` is unset. `databricks api`, `databricks auth token`, `databricks auth describe`, and bundle commands now resolve `[DEFAULT]` to `cfg.Profile = "DEFAULT"` whenever the section has a host. Previously the SDK silently loaded `[DEFAULT]`'s values but left `cfg.Profile` empty, which produced a host-URL OAuth cache key while `databricks auth login` stored tokens under the `"DEFAULT"` key. The mismatch was masked by plaintext's host-key dual-write but surfaced under secure storage as `cache: token not found` or `InvalidRefreshTokenError`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.
* Add `postgres_catalogs` resource to bind a Unity Catalog catalog to a Postgres database on a Lakebase Autoscaling branch ([#5265](https://github.com/databricks/cli/pull/5265)).
* engine/direct: Changes to state file now persisted to .wal file right away instead of being saved in the end ([#5149](https://github.com/databricks/cli/pull/5149))

### Dependency updates
