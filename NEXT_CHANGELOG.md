# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI

### Bundles
* `bundle summary` now reports the deployed name and URL for `synced_database_tables` (loaded from state) instead of the raw `${resources...}` reference when the configured name embeds another resource's name, matching the existing behavior of `postgres_synced_tables` ([#5639](https://github.com/databricks/cli/pull/5639)).
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).

### Dependency updates

### API Changes
