# NEXT CHANGELOG

## Release v0.259.0

### Notable Changes
* Error when the absolute path to `databricks.yml` contains a glob character. These are: `*`, `?`, `[`, `]` and `^`. If the path to the `databricks.yml` file on your local filesystem contains one of these characters, that could lead to incorrect computation of glob patterns for the `includes` block and might cause resources to be deleted. After this patch users will not be at risk for unexpected deletions due to this issue. ([#3096](https://github.com/databricks/cli/pull/3096))
* Diagnostics messages are no longer buffered to be printed at the end of command, flushed after every mutator ([#3175](https://github.com/databricks/cli/pull/3175))
* Diagnostics are now always rendered with forward slashes in file paths, even on Windows ([#3175](https://github.com/databricks/cli/pull/3175))
* "bundle summary" now prints diagnostics to stderr instead of stdout in text output mode ([#3175](https://github.com/databricks/cli/pull/3175))
* "bundle summary" no longer prints recommendations, it will only prints warnings and errors ([#3175](https://github.com/databricks/cli/pull/3175))

### Dependency updates

### CLI

### Bundles
* Fix default search location for whl artifacts ([#3184](https://github.com/databricks/cli/pull/3184)). This was a regression introduced in 0.255.0.

### API Changes
