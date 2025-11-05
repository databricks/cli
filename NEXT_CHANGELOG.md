# NEXT CHANGELOG

## Release v0.276.0

### Notable Changes

### CLI
* Remove previously added flags from the `jobs create` and `pipelines create` commands. ([#3870](https://github.com/databricks/cli/pull/3870))

### Dependency updates

### Bundles
* Add `default-minimal` template for users who want a clean slate without sample code ([#3885](https://github.com/databricks/cli/pull/3885))
* Updated the default-python template to follow the Lakeflow conventions: pipelines as source files, pyproject.toml ([#3712](https://github.com/databricks/cli/pull/3712)).
* Fix a permissions bug adding second IS\_OWNER and causing "The job must have exactly one owner." error. Introduced in 0.274.0. ([#3850](https://github.com/databricks/cli/pull/3850))

### API Changes
