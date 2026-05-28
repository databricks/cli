# NEXT CHANGELOG

## Release v1.2.0

### Notable Changes

### CLI
* `experimental open` now opens every DABs resource type that has a workspace URL, picking up `catalogs`, `schemas`, `volumes`, `database_instances`, `database_catalogs`, `synced_database_tables`, `postgres_catalogs`, `postgres_synced_tables`, `quality_monitors`, `vector_search_endpoints`, and `vector_search_indexes` ([#5346](https://github.com/databricks/cli/pull/5346)).

### Bundles
* Retry transient HTTP 504 Gateway Timeout errors in direct deployment engine ([#5349](https://github.com/databricks/cli/pull/5349)).
* Preserve `.designer.ipynb` suffix when translating notebook task paths so Lakeflow Designer files referenced from a `notebook_task` resolve correctly in the workspace ([#5370](https://github.com/databricks/cli/pull/5370)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.136.0 to v0.138.0 ([#5361](https://github.com/databricks/cli/pull/5361))

### API Changes
