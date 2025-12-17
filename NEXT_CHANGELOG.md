# NEXT CHANGELOG

## Release v0.281.0

### Notable Changes

### CLI
* Fix lakeview publish to default `embed_credentials` to false ([#4066](https://github.com/databricks/cli/pull/4066))

### Bundles
* Add support for configurable catalog/schema for dashboards ([#4130](https://github.com/databricks/cli/pull/4130))
* engine/direct: Fix dependency-ordered deletion by persisting depends_on in state ([#4105](https://github.com/databricks/cli/pull/4105))
* Pass SYSTEM_ACCESSTOKEN from env to the Terraform provider ([#4135](https://github.com/databricks/cli/pull/4135)
* `bundle deployment migrate`: when running `bundle plan` propagate `-var` arguments.

### Dependency updates
* Upgrade Go SDK to 0.94.0 ([#4148](https://github.com/databricks/cli/pull/4148))
* Upgrade Terraform provider to 1.100.0 ([#4150](https://github.com/databricks/cli/pull/4150))

### API Changes
