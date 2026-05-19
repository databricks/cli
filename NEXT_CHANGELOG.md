# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

* Breaking change: U2M (`auth_type = databricks-cli`) tokens are now stored in the OS-native secure store by default (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux) instead of `~/.databricks/token-cache.json`. After upgrading, run `databricks auth login` once per profile to re-authenticate; cached tokens from older versions are not migrated. To keep the previous file-backed storage, set `DATABRICKS_AUTH_STORAGE=plaintext` or add `auth_storage = plaintext` under `[__settings__]` in `~/.databrickscfg` (the env var takes precedence over the config setting), then re-run `databricks auth login`.

### CLI

* Added `databricks aitools` command group for installing Databricks skills into your coding agents (Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity). Skills are fetched from [github.com/databricks/databricks-agent-skills](https://github.com/databricks/databricks-agent-skills) and either symlinked into each agent's skills directory or copied into the current project. Use `databricks aitools install` to set up, `update` to pull newer versions, `list` to see what's available, and `uninstall` to remove them.
* `[__settings__].default_profile` is now consulted as a fallback by `databricks api`, `databricks auth token`, and bundle commands when neither `--profile` nor `DATABRICKS_CONFIG_PROFILE` is set. `databricks auth token` continues to give precedence to `DATABRICKS_HOST` over `default_profile`. For bundle commands, `default_profile` only applies when the bundle does not pin its own `workspace.host`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))
* Support `replace_existing: true` on `postgres_branches` and `postgres_endpoints` so bundles can manage the implicitly-created production branch and primary read-write endpoint of a Lakebase project.

### Dependency updates
