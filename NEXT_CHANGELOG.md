# NEXT CHANGELOG

## Release v0.252.0

### Notable Changes
* Add scripts to DABs. Users can now define and co-version their scripts in their bundles. These scripts will automatically be authenticated to the same credentials as the bundle itself. ([#2813](https://github.com/databricks/cli/pull/2813))

### Dependency updates
* Upgraded to TF provider 1.79.0 ([#2869](https://github.com/databricks/cli/pull/2869))

### CLI

### Bundles
* Removed unused fields from resources.models schema: creation\_timestamp, last\_updated\_timestamp, latest\_versions and user\_id. Using them now raises a warning.
* Preserve folder structure for app source code in bundle generate ([#2848](https://github.com/databricks/cli/pull/2848))
* Fixed normalising requirements file path in dependencies section ([#2861](https://github.com/databricks/cli/pull/2861))
* Fix default-python template not to add environments when serverless=yes and include\_python=no ([#2866](https://github.com/databricks/cli/pull/2866))
* Fixed handling of Unicode characters in Python support ([#2873](https://github.com/databricks/cli/pull/2873))

### API Changes
