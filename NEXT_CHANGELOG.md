# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI
* `workspace export-dir` no longer aborts when a workspace object's name is not a legal local filename (e.g. a notebook named `New Notebook 2026-05-04 13:54:24` whose `:` is illegal on Windows). Such files are now exported under a sanitized name with a warning and the export completes ([#5171](https://github.com/databricks/cli/issues/5171)).

### Bundles
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.141.0 to v0.147.0 ([#5636](https://github.com/databricks/cli/pull/5636)).

### API Changes
