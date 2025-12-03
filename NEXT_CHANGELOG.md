# NEXT CHANGELOG

## Release v0.279.0

### Notable Changes
* New deployment engine for DABs that does not require Terraform is available to try in experimental mode. Not recommended for production workloads yet. Documentation at https://github.com/databricks/cli/blob/main/docs/direct.md

### CLI

### Dependency updates

### Bundles
* Add support for alerts to DABs ([#4004](https://github.com/databricks/cli/pull/4004))
* Allow `file://` URIs in job libraries to reference runtime filesystem paths (e.g., JARs pre-installed on clusters via init scripts). These paths are no longer treated as local files to upload. ([#3884](https://github.com/databricks/cli/pull/3884))

### API Changes
