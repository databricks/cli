# NEXT CHANGELOG

## Release v0.262.0

### Notable Changes
* Breaking change: if the relative paths to the bundle resources are defined relatively to where the job or pipeline
is defined rather than the configuration file where this path is defined, DABs will return an error.
Previously, it would fallback to older path resolution. This fallback path resolution is deprecated now.
Please update the path to be relative to the configuration file where this path is defined.

### Dependency updates

### CLI

### Bundles
* [Breaking Change] Convert warning about using fallback paths to error ([#3225](https://github.com/databricks/cli/pull/3225))

### API Changes
