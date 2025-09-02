# NEXT CHANGELOG

## Release v0.267.0

### Notable Changes

### CLI
* Introduce retries to `databricks psql` command ([#3492](https://github.com/databricks/cli/pull/3492))
* Add rule files for coding agents working on the CLI code base ([#3245](https://github.com/databricks/cli/pull/3245))

### Dependency updates
* Upgrade TF provider to 1.88.0 ([#3529](https://github.com/databricks/cli/pull/3529))
* Upgrade Go SDK to 0.82.0

### Bundles
* Update default-python template to make DB Connect work out of the box for unit tests, using uv to install dependencies ([#3254](https://github.com/databricks/cli/pull/3254))
* Add support for `TaskRetryMode` for continuous jobs ([#3529](https://github.com/databricks/cli/pull/3529))
* Add support for specifying database instance as an application resource ([#3529](https://github.com/databricks/cli/pull/3529))

### API Changes
