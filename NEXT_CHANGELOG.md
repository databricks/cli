# NEXT CHANGELOG

## Release v0.299.2

### CLI

* `auth login` no longer falls back to plaintext when the OS keyring is reachable but locked. The unlock prompt shown by the probe now runs in parallel with the OAuth flow, and the token is stored in the keyring once the user has typed their password.
* `databricks auth describe` now reports where U2M (`databricks-cli`) tokens are stored: `plaintext` (`~/.databricks/token-cache.json`) or `secure` (OS keyring), and the source of the choice (env var, config setting, or default).
* Marked the default profile in the interactive pickers shown by `databricks auth switch`, `databricks auth logout`, `databricks auth token`, and `databricks auth login`, and moved it to the top of the list. `databricks auth login` and `databricks auth logout` now offer the same selectors as `databricks auth token` and `databricks auth switch` respectively.
* The interactive auth profile pickers now start in search mode so typing immediately filters the list, and the action entries (`+ Create a new profile`, `→ Enter a host URL manually`) are visually distinct from real profiles and stay visible regardless of the search query.

### Bundles
* Stop applying `presets.name_prefix` (and the dev-mode `[dev <user>]` rename) to `vector_search_endpoints` ([#5209](https://github.com/databricks/cli/pull/5209)).

* Fix `bundle generate` job to preserve nested notebook directory structure ([#4596](https://github.com/databricks/cli/pull/4596))
* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.

### Dependency updates

* Bump Go toolchain to 1.25.10 ([#5213](https://github.com/databricks/cli/pull/5213)).
