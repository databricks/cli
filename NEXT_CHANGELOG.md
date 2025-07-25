# NEXT CHANGELOG

## Release v0.266.0

### Notable Changes
* Breaking change: if the relative paths to the bundle resources are defined relatively to where the job or pipeline
is defined rather than the configuration file where this path is defined, DABs will return an error.
Previously, it would fallback to older path resolution. This fallback path resolution is deprecated now.
Please update the path to be relative to the configuration file where this path is defined.
* Add support volumes in Python support ([#3383])(https://github.com/databricks/cli/pull/3383))

### Dependency updates

### CLI

### Dependency updates

### Bundles
* [Breaking Change] Remove deprecated path fallback mechanism for jobs and pipelines ([#3225](https://github.com/databricks/cli/pull/3225))

### API Changes
