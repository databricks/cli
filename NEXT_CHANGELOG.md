# NEXT CHANGELOG

## Release v1.4.0

### Notable Changes

### CLI

### Bundles
* Remove DEVELOPMENT-stage API enum values and types from the `databricks-bundles` Python package; these were never accepted by the backend. This removes 19 `Privilege` values (e.g. `VIEW_OBJECT`) from catalogs, schemas and volumes, `HardwareAcceleratorType.GPU_1X_H100`, and `IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue` ([#5484](https://github.com/databricks/cli/pull/5484)).

### Dependency updates

### API Changes
