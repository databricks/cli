# NEXT CHANGELOG

## Release v0.247.0

### Notable Changes

### Dependency updates

### CLI
* Added include/exclude flags support to sync command ([#2650](https://github.com/databricks/cli/pull/2650))

### Bundles
* Added support for model serving endpoints in deployment bind/unbind commands ([#2634](https://github.com/databricks/cli/pull/2634))
* Added include/exclude flags support to bundle sync command ([#2650](https://github.com/databricks/cli/pull/2650))
* Added JSON schema for resource permissions ([#2674](https://github.com/databricks/cli/pull/2674))
* Removed pipeline 'deployment' field from jsonschema ([#2653](https://github.com/databricks/cli/pull/2653))
* Updated JSON schema for deprecated pipeline fields ([#2646](https://github.com/databricks/cli/pull/2646))
* The --config-dir and --source-dir flags for "bundle generate app" are now relative to CWD, not bundle root ([#2683](https://github.com/databricks/cli/pull/2683))
* Reverts [#2549](https://github.com/databricks/cli/pull/2549) to resolve issues with Web Terminal host mismatch ([#2685](https://github.com/databricks/cli/pull/2685))

### API Changes
