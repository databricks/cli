# NEXT CHANGELOG

## Release v0.286.0

### Notable Changes

### CLI

* Improve performance of `databricks fs cp` command by parallelizing file uploads when
  copying directories with the `--recursive` flag.
* Fix: Support trigger_pause_status preset in alerts ([#4323](https://github.com/databricks/cli/pull/4323))

### Bundles

* Add missing values to SchemaGrantPrivilege enum ([#4380](https://github.com/databricks/cli/pull/4380))

### Dependency updates

### API Changes
