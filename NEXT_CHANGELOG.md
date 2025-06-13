# NEXT CHANGELOG

## Release v0.256.0

### Notable Changes

### Dependency updates

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))
* Use OS aware runner instead of bash for run-local command ([#2996](https://github.com/databricks/cli/pull/2996))

### Bundles
* Fix reading dashboard contents when the sync root is different than the bundle root ([#3006](https://github.com/databricks/cli/pull/3006))
* When glob for wheels is used, like "\*.whl", it will filter out different version of the same package and will only take the most recent version. ([#2982](https://github.com/databricks/cli/pull/2982))
* When building Python artifacts as part of "bundle deploy" we no longer delete `dist`, `build`, `*egg-info` and `__pycache__` directories. ([#2982](https://github.com/databricks/cli/pull/2982))

### API Changes
