# NEXT CHANGELOG

## Release v0.248.0

### Notable Changes

### Dependency updates

### CLI

### Bundles
* Do not exit early when checking incompatible tasks for specified DBR ([#2692](https://github.com/databricks/cli/pull/2692))
* Removed include/exclude flags support from bundle sync command ([#2718](https://github.com/databricks/cli/pull/2718))
* Do not clean up Python artifacts dist and build folder in "bundle validate", do it in "bundle deploy". This reverts the behaviour introduced in 0.245.0 ([#2722](https://github.com/databricks/cli/pull/2722))
* Fixed a regression with pipeline library globs introduced in 0.247.0 ([#2723](https://github.com/databricks/cli/pull/2723))

### API Changes
