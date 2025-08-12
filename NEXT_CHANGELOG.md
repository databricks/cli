# NEXT CHANGELOG

## Release v0.264.0

### Notable Changes

### Dependency updates
* Upgrade TF provider to 1.86.0 ([#3374](https://github.com/databricks/cli/pull/3374))
* Update Go SDK to 0.79.0 ([#3376](https://github.com/databricks/cli/pull/3376))

### CLI
* Fixed panic when providing a CLI command with an incorrect JSON input ([#3398](https://github.com/databricks/cli/pull/3398))

### Bundles
* Changed logic for resolving `${resources...}` references. Previously this would be done by terraform at deploy time. Now if it references a field that is present in the config, it will be done by DABs during bundle loading ([#3370](https://github.com/databricks/cli/pull/3370))
* Add support for tagging pipelines ([#3086](https://github.com/databricks/cli/pull/3086))
* Add warning for when an invalid value is specified for an enum field ([#3050](https://github.com/databricks/cli/pull/3050))

### API Changes
