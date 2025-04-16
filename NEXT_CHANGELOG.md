# NEXT CHANGELOG

## Release v0.248.0

### Notable Changes
* Fixed a regression with pipeline library globs introduced in 0.247.0 ([#2723](https://github.com/databricks/cli/pull/2723)). The issue caused glob patterns to fail when using paths relative to a directory that is not the bundle root.

### Dependency updates
* Updated Go SDK to 0.63.0 ([#2716](https://github.com/databricks/cli/pull/2716))

### CLI
* Added an error when invalid subcommand is provided for CLI commands ([#2655](https://github.com/databricks/cli/pull/2655))
* Added dry-run flag support to sync command ([#2657](https://github.com/databricks/cli/pull/2657))

### Bundles
* Do not use app config section in test templates and generated app configuration ([#2599](https://github.com/databricks/cli/pull/2599))
* Do not exit early when checking incompatible tasks for specified DBR ([#2692](https://github.com/databricks/cli/pull/2692))
* Removed include/exclude flags support from bundle sync command ([#2718](https://github.com/databricks/cli/pull/2718))
* Do not clean up Python artifacts dist and build folder in "bundle validate", do it in "bundle deploy". This reverts the behaviour introduced in 0.245.0 ([#2722](https://github.com/databricks/cli/pull/2722))

### API Changes
* Added enable-export-notebook, enable-notebook-table-clipboard and enable-results-downloading workspace-level commands.
* Removed unused `timeout` and `no-wait` flags for clusters and pipelines
