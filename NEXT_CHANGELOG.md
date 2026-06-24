# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI

### Bundles
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).
* test: notebook: never strip the .designer.ipynb extension


### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.141.0 to v0.147.0 ([#5636](https://github.com/databricks/cli/pull/5636)).
* Bump Terraform provider from v1.117.0 to v1.118.0 ([#5637](https://github.com/databricks/cli/pull/5637)).

### API Changes
