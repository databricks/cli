# NEXT CHANGELOG

## Release v0.246.0

### Notable Changes
Previously ".internal" folder under artifact_path was not cleaned up as expected. In this release this behaviour is fixed and now DABs cleans up this folder before uploading artifacts to it.

### Dependency updates
* Bump golangci-lint version to v2.0.2 from v1.63.4 ([#2586](https://github.com/databricks/cli/pull/2586)).

### CLI
* Include supported values for enum positional args in docs ([#2557](https://github.com/databricks/cli/pull/2557))
* Upgrade Go SDK to 0.61.0 ([#2575](https://github.com/databricks/cli/pull/2575))
* Fix `databricks api` command usage string to include PATH positional argument ([#2591](https://github.com/databricks/cli/pull/2591))

### Bundles
* Fixed cleaning up artifact path .internal folder ([#2572](https://github.com/databricks/cli/pull/2572))
* Added support for quality monitors in deployment bind/unbind commands ([#2583](https://github.com/databricks/cli/pull/2583))
* Comment out email\_notifications section in builtin templates ([#2565](https://github.com/databricks/cli/pull/2565))
* New DATABRICKS_BUNDLE_RESTRICTED_CODE_EXECUTION environment variable that rejects running scripts and Python code when it is set ([#2598](https://github.com/databricks/cli/pull/2598))

### API Changes
