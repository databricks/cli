# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).

### Bundles
* Remove API enum values and types that are still in development from the `databricks-bundles` Python package; these were never accepted by the backend ([#5484](https://github.com/databricks/cli/pull/5484)).
* direct: Fix resolving a resource reference that is used more than once within the same field ([#5558](https://github.com/databricks/cli/pull/5558)).
* Bundle variable references now accept Unicode letters in path segments (e.g. `${var.变量}`). ([#5532](https://github.com/databricks/cli/pull/5532))
* Ignore remote changes for vector search direct_access_index_spec.schema_json to prevent drift when the backend normalizes the schema ([#5481](https://github.com/databricks/cli/pull/5481)).
* Remove hidden, never-functional `--existing-dashboard-id`, `--existing-dashboard-path`, `--existing-alert-id`, and `--existing-genie-space-id` alias flags from `bundle generate`; use the documented `--existing-id` / `--existing-path` flags instead ([#5591](https://github.com/databricks/cli/pull/5591)).
* engine/direct: Fix WAL corruption after two consecutive failed deploys ([#5606](https://github.com/databricks/cli/pull/5606)).
* engine/direct: Don't open the deployment state WAL when a deploy's plan fails ([#5607](https://github.com/databricks/cli/pull/5607)).
* Ignore unity catalog managed schema property defaults to avoid unnecessary drift ([#5195](https://github.com/databricks/cli/pull/5195)).
* Added an `env:` section to `scripts.<name>` for declaring environment variables that may reference `${bundle.*}`, `${workspace.*}`, and `${var.*}` ([#4179](https://github.com/databricks/cli/issues/4179)).
* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).

### Dependency updates

### API Changes
