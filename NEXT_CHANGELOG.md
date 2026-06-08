# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes
* The `direct` deployment engine is now Generally Available and the default for new deployments. To opt out, set `engine: terraform` under `bundle` in your `databricks.yml` or set `DATABRICKS_BUNDLE_ENGINE=terraform`. Existing deployments keep their current engine; see https://docs.databricks.com/aws/en/dev-tools/bundles/direct to migrate.

### CLI

### Bundles
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).

### Dependency updates

### API Changes
