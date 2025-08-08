# NEXT CHANGELOG

## Release v0.264.0

### Notable Changes

### Dependency updates
* Update Go SDK to 0.79.0 ([#3376](https://github.com/databricks/cli/pull/3376))

### CLI

### Bundles
* Changed logic for resolving `${resources...}` references. Previously this would be done by terraform at deploy time. Now if it references a field that is present in the config, it will be done by DABs during bundle loading ([#3370](https://github.com/databricks/cli/pull/3370))

### API Changes
