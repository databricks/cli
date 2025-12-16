# NEXT CHANGELOG

## Release v0.281.0

### Notable Changes

### CLI

* Improve performance of `databricks fs cp` command by parallelizing file uploads when
  copying directories with the `--recursive` flag.

### Bundles
* engine/direct: Fix dependency-ordered deletion by persisting depends_on in state ([#4105](https://github.com/databricks/cli/pull/4105))
* Pass SYSTEM_ACCESSTOKEN from env to the Terraform provider ([#4135](https://github.com/databricks/cli/pull/4135)

### Dependency updates

### API Changes
