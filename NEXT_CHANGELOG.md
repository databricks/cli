# NEXT CHANGELOG

## Release v0.256.0

### Notable Changes
* Add scripts to DABs. Users can now define and co-version their scripts in their bundles. These scripts will automatically be authenticated to the same credentials as the bundle itself. ([#2813](https://github.com/databricks/cli/pull/2813))

### Dependency updates

### CLI

### Bundles
* Fix reading dashboard contents when the sync root is different than the bundle root ([#3006](https://github.com/databricks/cli/pull/3006))
* When glob for wheels is used, like "\*.whl", it will filter out different version of the same package and will only take the most recent version. ([#2982](https://github.com/databricks/cli/pull/2982))
* When building Python artifacts as part of "bundle deploy" we no longer delete `dist`, `build`, `*egg-info` and `__pycache__` directories. ([#2982](https://github.com/databricks/cli/pull/2982))

### API Changes
