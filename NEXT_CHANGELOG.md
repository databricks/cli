# NEXT CHANGELOG

## Release v0.282.0

### Notable Changes
* engine/direct: New plan format (v2) ([#4201](https://github.com/databricks/cli/pull/4201))

### CLI
* Skip non-exportable objects (e.g., `MLFLOW_EXPERIMENT`) during `workspace export-dir` instead of failing ([#4101](https://github.com/databricks/cli/pull/4101))

### Bundles
* Allow `domain_friendly_name` to be used in `name_prefix` in development mode ([#4173](https://github.com/databricks/cli/pull/4173))
* Add missing schema grants privileges ([#4139](https://github.com/databricks/cli/pull/4139))
* Add support for `bundle generate alert` command ([#4108](https://github.com/databricks/cli/pull/4108))
* Add support for `.dbalert.json` files ([#3602](https://github.com/databricks/cli/pull/3602))
* Pass `SYSTEM_TEAMFOUNDATIONCOLLECTIONURI` from env to the Terraform provider for Azure DevOps OIDC auth ([#4169](https://github.com/databricks/cli/pull/4169))
* Add `ipykernel` to the `default` template to enable Databricks Connect notebooks in Cursor/VS Code ([#4164](https://github.com/databricks/cli/pull/4164))
* Add interactive SQL warehouse picker to `default-sql` and `dbt-sql` bundle templates ([#4170](https://github.com/databricks/cli/pull/4170))
* Add `name`, `target` and `mode` fields to the deployment metadata file ([#4180](https://github.com/databricks/cli/pull/4180))
* engine/direct: Fix app deployment failure when app is in `DELETING` state ([#4176](https://github.com/databricks/cli/pull/4176))
* engine/direct: Changes in config that match remote changes no longer trigger an update ([#4201](https://github.com/databricks/cli/pull/4201))

### Dependency updates

### API Changes
