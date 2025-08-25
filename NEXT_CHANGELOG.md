# NEXT CHANGELOG

## Release v0.266.0

### Notable Changes
* Breaking change: DABs now return an error when paths are incorrectly defined relative to the job or
pipeline definition location instead of the configuration file location. Previously, the CLI would show a
warning and fallback to resolving the path relative to the resource location. Users must update their bundle
configurations to define all relative paths relative to the configuration file where the path is specified.
See more details here: ([#3225](https://github.com/databricks/cli/pull/3225))
* Add support volumes in Python support ([#3383])(https://github.com/databricks/cli/pull/3383))

### Dependency updates

### CLI

### Dependency updates

### Bundles
* [Breaking Change] Remove deprecated path fallback mechanism for jobs and pipelines ([#3225](https://github.com/databricks/cli/pull/3225))
* Modified run flag validation to allow `--refresh` and `--full-refresh` flags to be used together ([#3300](https://github.com/databricks/cli/pull/3300))

### API Changes
