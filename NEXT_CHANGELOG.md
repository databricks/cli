# NEXT CHANGELOG

## Release v0.258.0

### Notable Changes
* Switch default-python template to use pyproject.toml + dynamic\_version in dev target. uv is now required. ([#3042](https://github.com/databricks/cli/pull/3042))

### Dependency updates
* Upgraded TF provider to 1.84.0 ([#3151](https://github.com/databricks/cli/pull/3151))

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))

### Bundles
* Added support for pipeline environment field ([#3153](https://github.com/databricks/cli/pull/3153))
* "bundle summary" now prints diagnostic warnings to stderr ([#3123](https://github.com/databricks/cli/pull/3123))
* "bundle open" will print the URL before opening the browser ([#3168](https://github.com/databricks/cli/pull/3168))

### API Changes
