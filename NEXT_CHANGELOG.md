# NEXT CHANGELOG

## Release v0.299.2

### CLI

* Promote the aitools skills-management surface (`install`, `update`, `uninstall`, `list`, `version`) from `databricks experimental aitools` to top-level `databricks aitools`. The old paths under `databricks experimental aitools` continue to work as silent backward-compat aliases. The `tools` subtree (`query`, `discover-schema`, `get-default-warehouse`, `statement …`) and the `skills` alias group remain under `databricks experimental aitools`.
* `databricks api` now works against unified hosts. Adds `--account` to scope a call to the account API and `--workspace-id` to override the workspace routing identifier per call. A `?o=<workspace-id>` query parameter on the path (the SPOG URL convention used by the Databricks UI) is also recognized as a per-call workspace override, so URLs pasted from the browser route correctly.
* JSON output for single objects now uses standard `"key": "value"` spacing (matching list output and `encoding/json` defaults).
* `databricks auth describe` now reports where U2M (`databricks-cli`) tokens are stored: `plaintext` (`~/.databricks/token-cache.json`) or `secure` (OS keyring), and the source of the choice (env var, config setting, or default).
* Marked the default profile in the interactive pickers shown by `databricks auth switch`, `databricks auth logout`, `databricks auth token`, and `databricks auth login`, and moved it to the top of the list. `databricks auth login` and `databricks auth logout` now offer the same selectors as `databricks auth token` and `databricks auth switch` respectively.

### Bundles
* Stop applying `presets.name_prefix` (and the dev-mode `[dev <user>]` rename) to `vector_search_endpoints` ([#5209](https://github.com/databricks/cli/pull/5209)).

* Fix `bundle generate` job to preserve nested notebook directory structure ([#4596](https://github.com/databricks/cli/pull/4596))
* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.

### Dependency updates

* Bump Go toolchain to 1.25.10 ([#5213](https://github.com/databricks/cli/pull/5213)).
