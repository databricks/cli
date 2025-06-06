# NEXT CHANGELOG

## Release v0.255.0

### Notable Changes

### Dependency updates

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. Also added a test to ensure `cluster_id` is retained after login. ([#2988](https://github.com/databricks/cli/pull/2988))
* Use OS aware runner instead of bash for run-local command ([#2996](https://github.com/databricks/cli/pull/2996))

### Bundles
* Fix "bundle summary -o json" to render null values properly ([#2990](https://github.com/databricks/cli/pull/2990))

### API Changes
