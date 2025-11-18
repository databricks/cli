# NEXT CHANGELOG

## Release v0.278.0

### Notable Changes

### CLI

### Dependency updates
* Bump Alpine Linux to 3.22 in the Docker image ([#3942](https://github.com/databricks/cli/pull/3942))

### Bundles
* Update templates to use serverless environment version 4 and matching Python version ([#3897](https://github.com/databricks/cli/pull/3897))
* Add a language prompt to the `default-minimal` template ([#3918](https://github.com/databricks/cli/pull/3918))
* Add `default-scala` template for Scala projects with SBT build configuration and example code ([#3906](https://github.com/databricks/cli/pull/3906))
* Allow `file://` URIs in job libraries to reference runtime filesystem paths (e.g., JARs pre-installed on clusters via init scripts). These paths are no longer treated as local files to upload. ([#3884](https://github.com/databricks/cli/pull/3884))

### API Changes
