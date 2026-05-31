# NEXT CHANGELOG

## Release v1.1.1

### Notable Changes

### CLI
* `experimental open` now opens every DABs resource type that has a workspace URL, picking up `catalogs`, `schemas`, `volumes`, `database_instances`, `database_catalogs`, `synced_database_tables`, `postgres_catalogs`, `postgres_synced_tables`, `quality_monitors`, `vector_search_endpoints`, and `vector_search_indexes` ([#5346](https://github.com/databricks/cli/pull/5346)).

### Bundles
* Retry transient HTTP 504 Gateway Timeout errors in direct deployment engine ([#5349](https://github.com/databricks/cli/pull/5349)).

### Dependency updates
* Bump `golang.org/x/crypto` from 0.51.0 to 0.52.0 ([#5344](https://github.com/databricks/cli/pull/5344)).
