# NEXT CHANGELOG

## Release v0.257.0

### Notable Changes

### Dependency updates

### CLI

### Bundles
* Remove support for deprecated `experimental/pydabs` config, use `experimental/python` instead. See [Configuration in Python
](https://docs.databricks.com/dev-tools/bundles/python). ([#3102](https://github.com/databricks/cli/pull/3102))
* Pass through OIDC token env variable to Terraform ([#3113](https://github.com/databricks/cli/pull/3113))

### API Changes
* Removed `databricks custom-llms` command group.
* Added `databricks ai-builder` command group.
* Added `databricks feature-store` command group.
