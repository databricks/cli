# NEXT CHANGELOG

## Release v1.1.0

### Notable Changes

### CLI

### Bundles
* The error reported when a direct-only resource (catalogs, external locations, vector search endpoints) is used with the terraform engine now also suggests setting `bundle.engine: direct` in `databricks.yml`, in addition to the `DATABRICKS_BUNDLE_ENGINE` environment variable ([#5295](https://github.com/databricks/cli/pull/5295)).
* Added an `env:` section to `scripts.<name>` for declaring environment variables whose values may reference `${bundle.*}`, `${workspace.*}`, and `${var.*}`. Script `content:` continues to be passed to the shell as-is (no DABs interpolation), avoiding ambiguity with shell variables. See issue [#4179](https://github.com/databricks/cli/issues/4179).
