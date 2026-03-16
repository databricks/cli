# NEXT CHANGELOG

## Release v0.295.0

### Notable Changes

- Add `bundle.engine` config setting to select the deployment engine (`terraform` or `direct`). The `DATABRICKS_BUNDLE_ENGINE` environment variable takes precedence over this setting. When the configured engine doesn't match existing deployment state, a warning is issued and the existing engine is used ([#4749](https://github.com/databricks/cli/pull/4749)).

### CLI

### Bundles
* engine/direct: Fix permanent drift on experiment name field ([#4627](https://github.com/databricks/cli/pull/4627))

### Dependency updates

### API Changes
