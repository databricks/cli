# NEXT CHANGELOG

## Release v0.257.0

### Notable Changes

### Dependency updates

### CLI

### Bundles
* Remove support for deprecated `experimental/pydabs` config, use `experimental/python` instead. See [Configuration in Python
](https://docs.databricks.com/dev-tools/bundles/python). ([#3102](https://github.com/databricks/cli/pull/3102))

* The `default-python` template now prompts if you want to use serverless compute (default to `yes`) ([#3051](https://github.com/databricks/cli/pull/3051)).

### API Changes
* Removed `databricks custom-llms` command group.
* Added `databricks ai-builder` command group.
* Added `databricks feature-store` command group.
