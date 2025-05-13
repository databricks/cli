# NEXT CHANGELOG

## Release v0.252.0

### Notable Changes

### Dependency updates
* Upgraded Go SDK to 0.69.0 ([#2867](https://github.com/databricks/cli/pull/2867))

### CLI

### Bundles
* Removed unused fields from resources.models schema: creation\_timestamp, last\_updated\_timestamp, latest\_versions and user\_id. Using them now raises a warning.
* Preserve folder structure for app source code in bundle generate ([#2848](https://github.com/databricks/cli/pull/2848))
* Fixed normalising requirements file path in dependencies section ([#2861](https://github.com/databricks/cli/pull/2861))

### API Changes
