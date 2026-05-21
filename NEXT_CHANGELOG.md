# NEXT CHANGELOG

## Release v1.1.0

### Notable Changes

### CLI

### Bundles
* The error reported when a direct-only resource (catalogs, external locations, vector search endpoints) is used with the terraform engine now also suggests setting `bundle.engine: direct` in `databricks.yml`, in addition to the `DATABRICKS_BUNDLE_ENGINE` environment variable.
