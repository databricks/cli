# NEXT CHANGELOG

## Release v0.293.0

### CLI
* Improve `auth token` error formatting for easier copy-paste of login commands ([#4602](https://github.com/databricks/cli/pull/4602))

### Bundles
* Modify grants to use SDK types ([#4666](https://github.com/databricks/cli/pull/4666))
* Modify permissions to use SDK types where available. This makes DABs validate permission levels, producing a warning on the unknown ones ([#4686](https://github.com/databricks/cli/pull/4686))

### Dependency updates
* Bump databricks-sdk-go from v0.112.0 to v0.119.0 ([#4631](https://github.com/databricks/cli/pull/4631), [#4695](https://github.com/databricks/cli/pull/4695))

### API Changes
