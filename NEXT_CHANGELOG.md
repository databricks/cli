# NEXT CHANGELOG

## Release v0.256.0

### Notable Changes

### Dependency updates

### CLI

### Bundles
* When building Python artifacts as part of "bundle deploy" we no longer delete `dist`, `build`, `*egg-info` and `__pycache__` directories ([#2982](https://github.com/databricks/cli/pull/2982))
* When glob for wheels is used, like "\*.whl", it will filter out different version of the same package and will only take the most recent version ([#2982](https://github.com/databricks/cli/pull/2982))
* Add preset `presets.artifacts_dynamic_version` that automatically enables `dynamic_version: true` on all "whl" artifacts ([#3074](https://github.com/databricks/cli/pull/3074))
* Update client version to "2" for the serverless variation of the default-python template ([#3083](https://github.com/databricks/cli/pull/3083))
* Fix reading dashboard contents when the sync root is different than the bundle root ([#3006](https://github.com/databricks/cli/pull/3006))
* Fix variable resolution for lookup variables with other references ([#3054](https://github.com/databricks/cli/pull/3054))
* Allow users to override the Terraform version to use by setting the `DATABRICKS_TF_VERSION` environment variable ([#3069](https://github.com/databricks/cli/pull/3069))

### API Changes
