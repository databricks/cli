# NEXT CHANGELOG

## Release v0.283.0

### Notable Changes
* Bundle commands now cache the user's account details to improve command latency.
To disable this, set the environment variable DATABRICKS_CACHE_ENABLED to false.

### CLI

### Bundles
* Pass SYSTEM_ACCESSTOKEN from env to the Terraform provider ([#4135](https://github.com/databricks/cli/pull/4135))
* Added missing schema grants privileges ([#4139](https://github.com/databricks/cli/pull/4139))
* Add `ipykernel` to the `default` template to enable Databricks Connect notebooks in Cursor/VS Code ([#4164](https://github.com/databricks/cli/pull/4164))
* Add interactive SQL warehouse picker to `default-sql` and `dbt-sql` bundle templates ([#4170](https://github.com/databricks/cli/pull/4170))
* Add `name`, `target` and `mode` fields to the deployment metadata file ([#4180](https://github.com/databricks/cli/pull/4180))
* Replace Black formatter with Ruff in Python bundle templates for faster, all-in-one linting and formatting ([#4196](https://github.com/databricks/cli/pull/4196))
* engine/direct: Fix app deployment failure when app is in `DELETING` state ([#4176](https://github.com/databricks/cli/pull/4176))
* engine/direct: Changes in config that match remote changes no longer trigger an update ([#4201](https://github.com/databricks/cli/pull/4201))
* Enable caching user identity by default ([#4202](https://github.com/databricks/cli/pull/4202))
* Pass additional Azure DevOps system variables ([#4236](https://github.com/databricks/cli/pull/4236))

### Dependency updates

### API Changes
