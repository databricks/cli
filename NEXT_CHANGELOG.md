# NEXT CHANGELOG

## Release v1.4.0

### Notable Changes

### CLI
* Show a once-per-day notice after a command when a newer CLI release is available, with a link to the release and the upgrade command for the detected install method. Suppressed for non-interactive/CI runs, JSON output, the Databricks Runtime, and development builds, and can be disabled with `DATABRICKS_CLI_DISABLE_UPDATE_CHECK` ([#5470](https://github.com/databricks/cli/pull/5470)).

### Bundles
* Remove API enum values and types that are still in development from the `databricks-bundles` Python package; these were never accepted by the backend ([#5484](https://github.com/databricks/cli/pull/5484)).
* direct: Fix resolving a resource reference that is used more than once within the same field ([#5558](https://github.com/databricks/cli/pull/5558)).
* Bundle variable references now accept Unicode letters in path segments (e.g. `${var.变量}`). ([#5532](https://github.com/databricks/cli/pull/5532))
* Ignore remote changes for vector search direct_access_index_spec.schema_json to prevent drift when the backend normalizes the schema ([#5481](https://github.com/databricks/cli/pull/5481)).
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them.

### Dependency updates

### API Changes
