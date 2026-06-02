# NEXT CHANGELOG

## Release v1.2.0

### Notable Changes

### CLI
* `experimental open` now opens every DABs resource type that has a workspace URL, picking up `catalogs`, `schemas`, `volumes`, `database_instances`, `database_catalogs`, `synced_database_tables`, `postgres_catalogs`, `postgres_synced_tables`, `quality_monitors`, `vector_search_endpoints`, and `vector_search_indexes` ([#5346](https://github.com/databricks/cli/pull/5346)).

### Bundles
* Retry transient HTTP 5xx and 408 errors in direct deployment engine ([#5349](https://github.com/databricks/cli/pull/5349), [#5364](https://github.com/databricks/cli/pull/5364)).
* Preserve `.designer.ipynb` suffix when translating notebook task paths so Lakeflow Designer files referenced from a `notebook_task` resolve correctly in the workspace ([#5370](https://github.com/databricks/cli/pull/5370)).
* Fix script output dropping last line without trailing newline ([#4995](https://github.com/databricks/cli/pull/4995)).
* Add Postgres role as a bundle resource (preview).

### Dependency updates
