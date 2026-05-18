# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* Added `databricks aitools` command group for installing Databricks skills into your coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity). Skills are fetched from [github.com/databricks/databricks-agent-skills](https://github.com/databricks/databricks-agent-skills) and either symlinked into each agent's skills directory or copied into the current project. Use `databricks aitools install` to set up, `update` to pull newer versions, `list` to see what's available, and `uninstall` to remove them.
* `[__settings__].default_profile` is now consulted as a fallback by `databricks api`, `databricks auth token`, and bundle commands when neither `--profile` nor `DATABRICKS_CONFIG_PROFILE` is set. `databricks auth token` continues to give precedence to `DATABRICKS_HOST` over `default_profile`. For bundle commands, `default_profile` only applies when the bundle does not pin its own `workspace.host`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))
||||||| original
<<<<<<< HEAD
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.
||||||| parent of c2fa9df0d (update NEXT_CHANGELOG.md)
=======
=======
<<<<<<< HEAD
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.
* Stop applying `presets.name_prefix` (and the dev-mode `[dev <user>]` rename) to `vector_search_endpoints` ([#5209](https://github.com/databricks/cli/pull/5209)).

* Fix `bundle generate` job to preserve nested notebook directory structure ([#4596](https://github.com/databricks/cli/pull/4596))
* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.
||||||| original
* Stop applying `presets.name_prefix` (and the dev-mode `[dev <user>]` rename) to `vector_search_endpoints` ([#5209](https://github.com/databricks/cli/pull/5209)).

* Fix `bundle generate` job to preserve nested notebook directory structure ([#4596](https://github.com/databricks/cli/pull/4596))
* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.
=======
* engine/direct: Changes to state file now persisted to .wal file right away instead of being saved in the end ([#5149](https://github.com/databricks/cli/pull/5149))
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.
<<<<<<< ours
* Add `postgres_catalogs` resource to bind a Unity Catalog catalog to a Postgres database on a Lakebase Autoscaling branch ([#5265](https://github.com/databricks/cli/pull/5265)).
>>>>>>> theirs

### Dependency updates
