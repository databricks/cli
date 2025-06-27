# NEXT CHANGELOG

## Release v0.258.0

### Notable Changes
* Error when the absolute path to `databricks.yml` contains a glob character. These are: `*`, `?`, `[`, `]` and `^`. If the path to the `databricks.yml` file on your local filesystem contains one of these characters, that could lead to incorrect computation of glob patterns for the `includes` block and might cause resources to be deleted. After this patch users will not be at risk for unexpected deletions due to this issue. ([#3096](https://github.com/databricks/cli/pull/3096))

### Dependency updates

### CLI
* Fixed an issue where running `databricks auth login` would remove the `cluster_id` field from profiles in `.databrickscfg`. The login process now preserves the `cluster_id` field. ([#2988](https://github.com/databricks/cli/pull/2988))

### Bundles
- "bundle summary" now prints diagnostic warnings to stderr ([#3123](https://github.com/databricks/cli/pull/3123))

### API Changes
