# NEXT CHANGELOG

## Release v0.273.0

### Notable Changes

### CLI

* Add the `--configure-serverless` flag to `databricks auth login` to configure Databricks Connect to use serverless.

### Dependency updates
* Upgrade Go SDK to 0.82.0 ([#3769](https://github.com/databricks/cli/pull/3769))
* Upgrade TF provider to 1.92.0 ([#3772](https://github.com/databricks/cli/pull/3772))

### Bundles
* Updated the internal lakeflow-pipelines template to use an "src" layout ([#3671](https://github.com/databricks/cli/pull/3671)).
* Fix for pip flags with equal sign being incorrectly treated as local file names ([#3766](https://github.com/databricks/cli/pull/3766))

### API Changes
