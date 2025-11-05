# NEXT CHANGELOG

## Release v0.277.0

### Notable Changes

### CLI

### Dependency updates

### Bundles
* Add validation that served_models and served_entities are not used at the same time. Add client side translation logic. ([#3880](https://github.com/databricks/cli/pull/3880))
* Fix handling of `file://` URIs in job libraries: `file://relative/path` uploads local files, while `file:///absolute/path` references runtime container assets. ([#3884](https://github.com/databricks/cli/pull/3884))

### API Changes
