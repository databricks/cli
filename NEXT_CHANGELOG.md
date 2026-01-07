# NEXT CHANGELOG

## Release v0.282.0

### Notable Changes

### CLI
* Skip non-exportable objects (e.g., `MLFLOW_EXPERIMENT`) during `workspace export-dir` instead of failing ([#4081](https://github.com/databricks/cli/issues/4081))
* Allow domain_friendly_name to be used in name_prefix in development mode ([#4173](https://github.com/databricks/cli/pull/4173))

### Bundles
* Pass SYSTEM_ACCESSTOKEN from env to the Terraform provider ([#4135](https://github.com/databricks/cli/pull/4135))
* Added missing schema grants privileges ([#4139](https://github.com/databricks/cli/pull/4139))
* Fix app deployment failure when app is in `DELETING` state ([#4176](https://github.com/databricks/cli/pull/4176))
* Add `ipykernel` to the `default` template to enable Databricks Connect notebooks in Cursor/VS Code ([#4164](https://github.com/databricks/cli/pull/4164))
* Add interactive SQL warehouse picker to `default-sql` and `dbt-sql` bundle templates ([#4170](https://github.com/databricks/cli/pull/4170))
* Add `name`, `target` and `mode` fields to the deployment metadata file ([#4180](https://github.com/databricks/cli/pull/4180))
* Enable caching user identity by default ([#4202](https://github.com/databricks/cli/pull/4202))

### Dependency updates

### API Changes
