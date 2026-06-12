# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes
* The `direct` deployment engine is now Generally Available and the default for new deployments. To opt out, set `engine: terraform` under `bundle` in your `databricks.yml` or set `DATABRICKS_BUNDLE_ENGINE=terraform`. Existing deployments keep their current engine; see https://docs.databricks.com/aws/en/dev-tools/bundles/direct to migrate.
* Introduced experimental support for X across all bundle resources.
  This continuation line documents the migration note for the same entry.

### CLI
* Added the `databricks quickstart` command, a short introduction to the CLI that prints a human-friendly guide interactively and an agent-oriented version when run non-interactively ([#5464](https://github.com/databricks/cli/pull/5464)).
* `databricks auth login` no longer prompts for workspace selection when logging in to an account console host (`https://accounts.*`). Pass `--workspace-id` explicitly to store a workspace ID on such a profile ([#5504](https://github.com/databricks/cli/pull/5504)).
* Added a `--output json` flag to `databricks auth describe` ([#5600](https://github.com/databricks/cli/pull/5600)).
* Fixed a panic in `databricks configure` when stdin is closed ([#5601](https://github.com/databricks/cli/pull/5601))
* Improved the `databricks quickstart` command output formatting.

### Bundles
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Mark vector search index index_subtype as backend_default to prevent drift after deployment ([#5454](https://github.com/databricks/cli/pull/5454)).
* `bundle deployment migrate`: handle resources added to or removed from `databricks.yml` since the last Terraform deploy ([#5463](https://github.com/databricks/cli/pull/5463)).
* First bundles change in this PR ([#5610](https://github.com/databricks/cli/pull/5610)).
* Second bundles change in the same PR ([#5610](https://github.com/databricks/cli/pull/5610)).
* Added `foo` support to bundle templates.

### Dependency updates
* Bump the Go SDK to v0.142.0 ([#5620](https://github.com/databricks/cli/pull/5620)).

### API Changes
* Added `Foo.bar` field on the Jobs API ([#5630](https://github.com/databricks/cli/pull/5630)).
