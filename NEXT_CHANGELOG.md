# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI

### Bundles
* Add documentation for the common bundle resource fields `permissions`, `lifecycle`, and `grants` in the JSON schema, so they surface in editor completions and the docs.
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).
* Fix `bundle deploy --plan` dropping a `postgres_role`'s `role_id`, which caused the role to be recreated on the next deploy ([#5672](https://github.com/databricks/cli/pull/5672)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.141.0 to v0.147.0 ([#5636](https://github.com/databricks/cli/pull/5636)).
* Bump Terraform provider from v1.117.0 to v1.118.0 ([#5637](https://github.com/databricks/cli/pull/5637)).

### API Changes
