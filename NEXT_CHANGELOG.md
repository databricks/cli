# NEXT CHANGELOG

## Release v1.2.0

### Notable Changes

### CLI
* `experimental open` now opens every DABs resource type that has a workspace URL, picking up `catalogs`, `schemas`, `volumes`, `database_instances`, `database_catalogs`, `synced_database_tables`, `postgres_catalogs`, `postgres_synced_tables`, `quality_monitors`, `vector_search_endpoints`, and `vector_search_indexes` ([#5346](https://github.com/databricks/cli/pull/5346)).

### Bundles
* engine/direct: Add declarative `bind` blocks under a target to bring existing workspace resources under bundle management at deploy time, with `bind` and `bind_and_update` actions surfaced in `bundle plan` output ([#4630](https://github.com/databricks/cli/pull/4630)).

### Dependency updates
