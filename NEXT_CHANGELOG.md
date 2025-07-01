# NEXT CHANGELOG

## Release v0.258.0

### Notable Changes

### Dependency updates
* Upgraded TF provider to 1.84.0 ([#3151](https://github.com/databricks/cli/pull/3151))

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))

### Bundles
* "bundle summary" now prints diagnostic warnings to stderr ([#3123](https://github.com/databricks/cli/pull/3123))

### API Changes
