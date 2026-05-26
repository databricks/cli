# NEXT CHANGELOG

## Release v1.1.0

### Notable Changes

### CLI

### Bundles
* The error reported when a direct-only resource (catalogs, external locations, vector search endpoints) is used with the terraform engine now also suggests setting `bundle.engine: direct` in `databricks.yml`, in addition to the `DATABRICKS_BUNDLE_ENGINE` environment variable ([#5295](https://github.com/databricks/cli/pull/5295)).

### Dependency updates

* Bump Go toolchain to 1.26.3 ([#5302](https://github.com/databricks/cli/pull/5302)).
* Bump `github.com/databricks/databricks-sdk-go` from v0.132.0 to v0.136.0.
