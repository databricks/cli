# NEXT CHANGELOG

## Release v0.283.0

### Notable Changes
* Bundle commands now cache the user's account details to improve command latency.
To disable this, set the environment variable DATABRICKS_CACHE_ENABLED to false.

### CLI

* Improve performance of `databricks fs cp` command by parallelizing file uploads when
  copying directories with the `--recursive` flag.

### Bundles
* Enable caching user identity by default ([#4202](https://github.com/databricks/cli/pull/4202))

### Dependency updates

### API Changes
