# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

### Bundles
* Added a `cascade_on_destroy` field to the pipeline resource to control whether destroying a pipeline also deletes its datasets (MVs, STs, Views). When unset, the server default applies; set `cascade_on_destroy: false` to retain the datasets on destroy. Supported with the direct deployment engine.

### Dependency updates

### API Changes
