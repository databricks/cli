# NEXT CHANGELOG

## Release v0.257.0

### Notable Changes
* Error when the absolute path to `databricks.yml` contains a glob character. These are: `*`, `?`, `[`, `]` and `^`. If the path to the `databricks.yml` file on your local filesystem contains one of these characters, that could lead to incorrect computation of glob patterns for the `includes` block and might cause resources to be deleted. After this patch users will not be at risk for unexpected deletions due to this issue. ([#3096](https://github.com/databricks/cli/pull/3096))

### Dependency updates

### CLI

### Bundles

### API Changes
