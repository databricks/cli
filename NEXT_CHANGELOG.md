# NEXT CHANGELOG

## Release v0.252.0

### Notable Changes

### Dependency updates
* Upgraded Go SDK to 0.69.0 ([#2867](https://github.com/databricks/cli/pull/2867))
* Upgraded to TF provider 1.79.0 ([#2869](https://github.com/databricks/cli/pull/2869))

### CLI

### Bundles
* Removed unused fields from resources.models schema: creation\_timestamp, last\_updated\_timestamp, latest\_versions and user\_id. Using them now raises a warning.
* Preserve folder structure for app source code in bundle generate ([#2848](https://github.com/databricks/cli/pull/2848))
* Fixed normalising requirements file path in dependencies section ([#2861](https://github.com/databricks/cli/pull/2861))
* Fix default-python template not to add environments when serverless=yes and include\_python=no ([#2866](https://github.com/databricks/cli/pull/2866))
* Fixed handling of Unicode characters in Python support ([#2873](https://github.com/databricks/cli/pull/2873))

### API Changes
