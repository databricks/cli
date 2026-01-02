# NEXT CHANGELOG

## Release v0.282.0

### Notable Changes

### CLI
* Skip non-exportable objects (e.g., `MLFLOW_EXPERIMENT`) during `workspace export-dir` instead of failing ([#4081](https://github.com/databricks/cli/issues/4081))

### Bundles
* Fix app deployment failure when app is in `DELETING` state ([#4176](https://github.com/databricks/cli/pull/4176))
* Add `ipykernel` to the `default` template to enable Databricks Connect notebooks in Cursor/VS Code ([#4164](https://github.com/databricks/cli/pull/4164))
* Add interactive SQL warehouse picker to `default-sql` and `dbt-sql` bundle templates ([#4170](https://github.com/databricks/cli/pull/4170))

### Dependency updates

### API Changes
