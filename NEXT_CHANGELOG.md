# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes
* The `direct` deployment engine is now Generally Available and the default for new deployments. To opt out, set `engine: terraform` under `bundle` in your `databricks.yml` or set `DATABRICKS_BUNDLE_ENGINE=terraform`. Existing deployments keep their current engine; see https://docs.databricks.com/aws/en/dev-tools/bundles/direct to migrate.

### CLI
* Added the `databricks quickstart` command, a short introduction to the CLI that prints a human-friendly guide interactively and an agent-oriented version when run non-interactively ([#5464](https://github.com/databricks/cli/pull/5464)).

### Bundles
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Mark vector search index index_subtype as backend_default to prevent drift after deployment ([#5454](https://github.com/databricks/cli/pull/5454)).
* `bundle deployment migrate`: handle resources added to or removed from `databricks.yml` since the last Terraform deploy ([#5463](https://github.com/databricks/cli/pull/5463)).
* Add the `genie_spaces` bundle resource for managing Databricks Genie spaces as code, plus `bundle generate genie-space` to import an existing space. Direct deployment engine only ([#5282](https://github.com/databricks/cli/pull/5282)).

### Dependency updates

### API Changes
