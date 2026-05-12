# NEXT CHANGELOG

## Release v0.299.2

### Notable Changes
* Breaking change: `vector_search_endpoints` renamed `min_qps` to `target_qps` in DABs configuration and the `vector-search-endpoints` commands, following the SDK rename in v0.131.0. Update any `databricks.yml` using `min_qps:` to `target_qps:` and any CLI invocations using `--min-qps` to `--target-qps`.

### CLI

* `databricks auth describe` now reports where U2M (`databricks-cli`) tokens are stored: `plaintext` (`~/.databricks/token-cache.json`) or `secure` (OS keyring), and the source of the choice (env var, config setting, or default).
* Marked the default profile in the interactive pickers shown by `databricks auth switch`, `databricks auth logout`, `databricks auth token`, and `databricks auth login`, and moved it to the top of the list. `databricks auth login` and `databricks auth logout` now offer the same selectors as `databricks auth token` and `databricks auth switch` respectively.

### Bundles
* Stop applying `presets.name_prefix` (and the dev-mode `[dev <user>]` rename) to `vector_search_endpoints` ([#5209](https://github.com/databricks/cli/pull/5209)).

* Fix `bundle generate` job to preserve nested notebook directory structure ([#4596](https://github.com/databricks/cli/pull/4596))
* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.

### Dependency updates

* Bump Go toolchain to 1.25.10 ([#5213](https://github.com/databricks/cli/pull/5213)).
* Bump `github.com/databricks/databricks-sdk-go` from v0.128.0 to v0.132.0.
