# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes
* The `direct` deployment engine is now Generally Available and the default for new deployments. To opt out, set `engine: terraform` under `bundle` in your `databricks.yml` or set `DATABRICKS_BUNDLE_ENGINE=terraform`. Existing deployments keep their current engine; see https://docs.databricks.com/aws/en/dev-tools/bundles/direct to migrate.

### CLI
* Added the `databricks quickstart` command, a short introduction to the CLI that prints a human-friendly guide interactively and an agent-oriented version when run non-interactively ([#5464](https://github.com/databricks/cli/pull/5464)).
* Add `databricks version --check` to report whether a newer CLI version is available and print the upgrade command for the detected install method ([#5469](https://github.com/databricks/cli/pull/5469)).
* Show a once-per-day notice after a command when a newer CLI release is available, with a link to the release and the upgrade command for the detected install method. Suppressed for non-interactive/CI runs, JSON output, the Databricks Runtime, and development builds, and can be disabled with `DATABRICKS_CLI_DISABLE_UPDATE_CHECK` ([#5470](https://github.com/databricks/cli/pull/5470)).

### Bundles
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Mark vector search index index_subtype as backend_default to prevent drift after deployment ([#5454](https://github.com/databricks/cli/pull/5454)).
* `bundle deployment migrate`: handle resources added to or removed from `databricks.yml` since the last Terraform deploy ([#5463](https://github.com/databricks/cli/pull/5463)).

### Dependency updates

### API Changes
