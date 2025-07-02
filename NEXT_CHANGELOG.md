# NEXT CHANGELOG

## Release v0.258.0

### Notable Changes
* Add scripts to DABs. Users can now define and co-version their scripts in their bundles. These scripts will automatically be authenticated to the same credentials as the bundle itself. ([#2813](https://github.com/databricks/cli/pull/2813))

### Dependency updates
* Upgraded TF provider to 1.84.0 ([#3151](https://github.com/databricks/cli/pull/3151))

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))

### Bundles
* Added support for pipeline environment field ([#3153](https://github.com/databricks/cli/pull/3153))
* "bundle summary" now prints diagnostic warnings to stderr ([#3123](https://github.com/databricks/cli/pull/3123))

### API Changes
