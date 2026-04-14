# NEXT CHANGELOG

## Release v0.297.0

### Notable Changes

### CLI
* Auth commands now accept a profile name as a positional argument ([#4840](https://github.com/databricks/cli/pull/4840))

* Add `auth logout` command for clearing cached OAuth tokens and optionally removing profiles ([#4613](https://github.com/databricks/cli/pull/4613), [#4616](https://github.com/databricks/cli/pull/4616), [#4647](https://github.com/databricks/cli/pull/4647))

### Bundles
* Added support for lifecycle.started option for apps ([#4672](https://github.com/databricks/cli/pull/4672))
* engine/direct: Fix permissions for resources.models ([#4941](https://github.com/databricks/cli/pull/4941))
* Fix resource references not correctly resolved in apps config section ([#4964](https://github.com/databricks/cli/pull/4964))
* Allow run_as for dashboards with embed_credentials set to false ([#4961](https://github.com/databricks/cli/pull/4961))
* direct: Pass changed fields into update mask for apps instead of wildcard ([#4963](https://github.com/databricks/cli/pull/4963))

### Dependency updates

### API Changes
