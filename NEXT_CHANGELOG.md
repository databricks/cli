# NEXT CHANGELOG

## Release v0.257.0

### Notable Changes

### Dependency updates

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))

### Bundles
* Remove support for deprecated `experimental/pydabs` config, use `experimental/python` instead. See [Configuration in Python
](https://docs.databricks.com/dev-tools/bundles/python). ([#3102](https://github.com/databricks/cli/pull/3102))

### API Changes
* Removed `databricks custom-llms` command group.
* Added `databricks ai-builder` command group.
* Added `databricks feature-store` command group.
