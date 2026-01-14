# NEXT CHANGELOG

## Release v0.283.0

### Notable Changes
* Bundle commands now cache the user's account details to improve command latency.
To disable this, set the environment variable DATABRICKS_CACHE_ENABLED to false.

### CLI
* Add commands to pipelines command group ([#4275](https://github.com/databricks/cli/pull/4275))
* Add support for unified host with experimental flag ([#4260](https://github.com/databricks/cli/pull/4260))

### Bundles
* Add support for configuring app.yaml options for apps via bundle config ([#4271](https://github.com/databricks/cli/pull/4271))
* Enable caching user identity by default ([#4202](https://github.com/databricks/cli/pull/4202))
* Do not show single node warning when is_single_node option is explicitly set ([#4272](https://github.com/databricks/cli/pull/4272))
* Fix false positive folder permission warnings and make them more actionable ([#4216](https://github.com/databricks/cli/pull/4216))
* Pass additional Azure DevOps system variables ([#4236](https://github.com/databricks/cli/pull/4236))
* Replace Black formatter with Ruff in Python bundle templates for faster, all-in-one linting and formatting ([#4196](https://github.com/databricks/cli/pull/4196))
* engine/direct: support quality monitors ([#4278](https://github.com/databricks/cli/pull/4278))

### Dependency updates

### API Changes
