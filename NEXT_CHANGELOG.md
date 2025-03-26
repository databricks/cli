# NEXT CHANGELOG

## Release v0.245.0

### Notable Changes
Previously ".internal" folder under artifact_path was not cleaned up as expected. In this release this behaviour is fixed and now DABs cleans up this folder before uploading artifacts to it.

### CLI

### Bundles
* Processing 'artifacts' section is now done in "bundle validate" (adding defaults, inferring "build", asserting required fields) ([#2526])(https://github.com/databricks/cli/pull/2526))
* When uploading artifacts, include relative path in log message ([#2539])(https://github.com/databricks/cli/pull/2539))
* Added support for clusters in deployment bind/unbind commands ([#2536](https://github.com/databricks/cli/pull/2536))
* Added support for volumes in deployment bind/unbind commands ([#2527](https://github.com/databricks/cli/pull/2527))
* Added support for dashboards in deployment bind/unbind commands ([#2516](https://github.com/databricks/cli/pull/2516))
* Added support for registered models in deployment bind/unbind commands ([#2556](https://github.com/databricks/cli/pull/2556))
* Added a mismatch check when host is defined in config and as an env variable ([#2549](https://github.com/databricks/cli/pull/2549))
* Fixed cleaning up artifact path .internal folder ([#2572](https://github.com/databricks/cli/pull/2572))

### API Changes
