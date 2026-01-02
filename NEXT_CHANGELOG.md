# NEXT CHANGELOG

## Release v0.282.0

### Notable Changes

### CLI
* Skip non-exportable objects (e.g., `MLFLOW_EXPERIMENT`) during `workspace export-dir` instead of failing ([#4081](https://github.com/databricks/cli/issues/4081))

### Bundles
* Add `ipykernel` to the `default` template to enable Databricks Connect notebooks in Cursor/VS Code ([#4164](https://github.com/databricks/cli/pull/4164))
* Add interactive SQL warehouse picker to `default-sql` and `dbt-sql` bundle templates ([#4170](https://github.com/databricks/cli/pull/4170))
* Add `name`, `target` and `mode` fields to the deployment metadata file ([#4180](https://github.com/databricks/cli/pull/4180))

### Dependency updates

### API Changes
