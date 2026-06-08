# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes

### CLI

### Bundles
* Retry transient HTTP 5xx and 408 errors in direct deployment engine ([#5349](https://github.com/databricks/cli/pull/5349), [#5364](https://github.com/databricks/cli/pull/5364)).
* Preserve `.designer.ipynb` suffix when translating notebook task paths so Lakeflow Designer files referenced from a `notebook_task` resolve correctly in the workspace ([#5370](https://github.com/databricks/cli/pull/5370)).
* Fix script output dropping last line without trailing newline ([#4995](https://github.com/databricks/cli/pull/4995)).
* Add Postgres role as a bundle resource (preview).
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).

### Dependency updates

### API Changes
