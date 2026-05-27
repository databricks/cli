# NEXT CHANGELOG

## Release v1.1.0

### Notable Changes

### CLI
* `experimental open` now opens every DABs resource type that has a workspace URL, picking up `catalogs`, `schemas`, `volumes`, `database_instances`, `database_catalogs`, `synced_database_tables`, `postgres_catalogs`, `postgres_synced_tables`, `quality_monitors`, `vector_search_endpoints`, and `vector_search_indexes` ([#5346](https://github.com/databricks/cli/pull/5346)).

### Bundles
* The error reported when a direct-only resource (catalogs, external locations, vector search endpoints) is used with the terraform engine now also suggests setting `bundle.engine: direct` in `databricks.yml`, in addition to the `DATABRICKS_BUNDLE_ENGINE` environment variable ([#5295](https://github.com/databricks/cli/pull/5295)).

* Added `vector_search_indexes` as a bundle resource (direct engine only). Supports UC grants and prompts for confirmation on recreate or delete since both are destructive ([#5123](https://github.com/databricks/cli/pull/5123)).

### Dependency updates

* Bump Go toolchain to 1.26.3 ([#5302](https://github.com/databricks/cli/pull/5302)).
* Bump `github.com/databricks/databricks-sdk-go` from v0.132.0 to v0.136.0.
