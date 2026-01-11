# NEXT CHANGELOG

## Release v0.283.0

### Notable Changes
* Bundle commands now cache the user's account details to improve command latency.
To disable this, set the environment variable DATABRICKS_CACHE_ENABLED to false.

### CLI

### Bundles
* Enable caching user identity by default ([#4202](https://github.com/databricks/cli/pull/4202))
* Replace Black formatter with Ruff in Python bundle templates for faster, all-in-one linting and formatting ([#4196](https://github.com/databricks/cli/pull/4196))
* Pass additional Azure DevOps system variables ([#4236](https://github.com/databricks/cli/pull/4236))

### Dependency updates

### API Changes
