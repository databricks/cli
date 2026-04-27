# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.
* Remove the `--experimental-is-unified-host` flag and stop reading `experimental_is_unified_host` from `.databrickscfg` profiles and the `DATABRICKS_EXPERIMENTAL_IS_UNIFIED_HOST` env var. Unified hosts are now detected exclusively from `/.well-known/databricks-config` discovery. The `experimental_is_unified_host` field is retained as a no-op in `databricks.yml` for schema compatibility.
* Added interactive pagination for list commands that have a row template (jobs, clusters, apps, pipelines, etc.). When stdin, stdout, and stderr are all TTYs, `databricks <resource> list` now streams 50 rows at a time and prompts `[space] more  [enter] all  [q|esc] quit`. ENTER can be interrupted by `q`/`esc`/`Ctrl+C` between pages. Colors and alignment match the existing non-paged output; column widths stay stable across pages. Piped output and `--output json` are unchanged.
* Added experimental OS-native secure token storage opt-in via `DATABRICKS_AUTH_STORAGE=secure`. Legacy file-backed token storage remains the default.
* Renamed the experimental `legacy` storage mode to `plaintext` and made it the resolver default. `DATABRICKS_AUTH_STORAGE=plaintext` and `[__settings__] auth_storage = plaintext` now select the file-backed token cache with host-key dual-write. The `legacy` keyword is no longer recognized (was undocumented; users on the default were unaffected).


### Bundles

### Dependency updates

* Added `github.com/zalando/go-keyring` as a new dependency (dormant until a later release enables experimental secure-storage for OAuth tokens).
