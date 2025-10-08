# NEXT CHANGELOG

## Release v0.272.0

### Notable Changes

### CLI

### Dependency updates

### Bundles
* Fix processing short pip flags in environment dependencies ([#3708](https://github.com/databricks/cli/pull/3708))
* Add support for referencing local files in -e pip flag for environment dependencies ([#3708](https://github.com/databricks/cli/pull/3708))
* Add error for when an etag is specified in dashboard configuration. Setting etags was never supported / valid in bundles but now users will see this error during validation rather than deployment. ([#3723](https://github.com/databricks/cli/pull/3723))
* Fix PIP flag processing in pipeline environment dependencies ([#3734](https://github.com/databricks/cli/pull/3734))

### API Changes
