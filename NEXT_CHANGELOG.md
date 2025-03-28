# NEXT CHANGELOG

## Release v0.246.0

### Notable Changes
Previously ".internal" folder under artifact_path was not cleaned up as expected. In this release this behaviour is fixed and now DABs cleans up this folder before uploading artifacts to it.

### CLI
* Upgrade Go SDK to 0.61.0 ([#2575](https://github.com/databricks/cli/pull/2575))

### Bundles
* Fixed cleaning up artifact path .internal folder ([#2572](https://github.com/databricks/cli/pull/2572))

### API Changes
