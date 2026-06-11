# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes

### CLI

### Bundles
* Remove DEVELOPMENT-stage API enum values and types from the `databricks-bundles` Python package; these were never accepted by the backend. This removes 19 `Privilege` values (e.g. `VIEW_OBJECT`) from catalogs, schemas and volumes, `HardwareAcceleratorType.GPU_1X_H100`, and `IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue` ([#5484](https://github.com/databricks/cli/pull/5484)).
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Mark vector search index index_subtype as backend_default to prevent drift after deployment ([#5454](https://github.com/databricks/cli/pull/5454)).
* `bundle deployment migrate`: handle resources added to or removed from `databricks.yml` since the last Terraform deploy ([#5463](https://github.com/databricks/cli/pull/5463)).

### Dependency updates

### API Changes
