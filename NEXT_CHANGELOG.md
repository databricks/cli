# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.
* Remove the `--experimental-is-unified-host` flag and stop reading `experimental_is_unified_host` from `.databrickscfg` profiles and the `DATABRICKS_EXPERIMENTAL_IS_UNIFIED_HOST` env var. Unified hosts are now detected exclusively from `/.well-known/databricks-config` discovery. The `experimental_is_unified_host` field is retained as a no-op in `databricks.yml` for schema compatibility.
* Added interactive pagination for list commands that have a row template (jobs, clusters, apps, pipelines, etc.). When stdin, stdout, and stderr are all TTYs, `databricks <resource> list` now streams 50 rows at a time and prompts `[space] more  [enter] all  [q|esc] quit`. ENTER can be interrupted by `q`/`esc`/`Ctrl+C` between pages. Colors and alignment match the existing non-paged output; column widths stay stable across pages. Piped output and `--output json` are unchanged.
* Added experimental OS-native secure token storage opt-in via `DATABRICKS_AUTH_STORAGE=secure`. Legacy file-backed token storage remains the default.
* Fixed a panic in `databricks warehouses update-default-warehouse-override` when invoked without all required positional arguments (e.g. picking a warehouse from the interactive drop-down and then hitting an index-out-of-range crash). The command now validates arguments up front and returns a usage error. Fixes [#5070](https://github.com/databricks/cli/issues/5070) via [#5079](https://github.com/databricks/cli/pull/5079).


### Bundles

* Translate relative paths in `alert_task.workspace_path` on job tasks to fully qualified workspace paths, matching the behavior of other task path fields. Applies to both regular tasks and `for_each_task` nested tasks ([#4836](https://github.com/databricks/cli/pull/4836)).
* Fix script output dropping last line without trailing newline ([#4995](https://github.com/databricks/cli/pull/4995))

### Dependency updates

* Added `github.com/zalando/go-keyring` as a new dependency (dormant until a later release enables experimental secure-storage for OAuth tokens).
