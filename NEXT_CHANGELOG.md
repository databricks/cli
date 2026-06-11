# NEXT CHANGELOG

## Release v1.4.0

### Notable Changes

### CLI
* Show a once-per-day notice after a command when a newer CLI release is available, with a link to the release and the upgrade command for the detected install method. Suppressed for non-interactive/CI runs, JSON output, the Databricks Runtime, and development builds, and can be disabled with `DATABRICKS_CLI_DISABLE_UPDATE_CHECK` ([#5470](https://github.com/databricks/cli/pull/5470)).

### Bundles
* Remove API enum values and types that are still in development from the `databricks-bundles` Python package; these were never accepted by the backend ([#5484](https://github.com/databricks/cli/pull/5484)).

### Dependency updates

### API Changes
