# NEXT CHANGELOG

## Release v0.252.0

### Notable Changes
* Add scripts to DABs. Users can now define and co-version their scripts in their bundles. These scripts will automatically be authenticated to the same credentials as the bundle itself. ([#2813](https://github.com/databricks/cli/pull/2813))

### Dependency updates

### CLI

### Bundles
* Removed unused fields from resources.models schema: creation\_timestamp, last\_updated\_timestamp, latest\_versions and user\_id. Using them now raises a warning.

### API Changes
