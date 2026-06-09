# NEXT CHANGELOG

## Release v1.3.0

### Notable Changes

### CLI
* Recommend upgrading when a newer CLI release is available ([#5482](https://github.com/databricks/cli/pull/5482)).

### Bundles
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Mark vector search index index_subtype as backend_default to prevent drift after deployment ([#5454](https://github.com/databricks/cli/pull/5454)).
* `bundle deployment migrate`: handle resources added to or removed from `databricks.yml` since the last Terraform deploy ([#5463](https://github.com/databricks/cli/pull/5463)).

### Dependency updates

### API Changes
