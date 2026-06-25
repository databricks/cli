# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

### Bundles

 * Fixed cluster resize failing with `INVALID_STATE` when the cluster terminates between plan and apply time. Resize is now always attempted via `clusters/resize` first, with an automatic fallback to `clusters/edit` if the cluster is not running ([#5716](https://github.com/databricks/cli/pull/5716)).

### Dependency updates

### API Changes
