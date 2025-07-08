# NEXT CHANGELOG

## Release v0.259.0

### Notable Changes
* Add support for arbitrary scripts in DABs. Users can now define scripts in their bundle configuration. These scripts automatically inherit the bundle's credentials for authentication. They can be invoked with the `bundle run` command. ([#2813](https://github.com/databricks/cli/pull/2813))
* Diagnostics messages are no longer buffered to be printed at the end of command, flushed after every mutator ([#3175](https://github.com/databricks/cli/pull/3175))
* Diagnostics are now always rendered with forward slashes in file paths, even on Windows ([#3175](https://github.com/databricks/cli/pull/3175))
* "bundle summary" now prints diagnostics to stderr instead of stdout in text output mode ([#3175](https://github.com/databricks/cli/pull/3175))
* "bundle summary" no longer prints recommendations, it will only prints warnings and errors ([#3175](https://github.com/databricks/cli/pull/3175))

### Dependency updates

### CLI

### Bundles
* Fix default search location for whl artifacts ([#3184](https://github.com/databricks/cli/pull/3184)). This was a regression introduced in 0.255.0.

### API Changes
