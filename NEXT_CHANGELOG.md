# NEXT CHANGELOG

## Release v0.295.0

### Notable Changes

- Add `bundle.engine` config setting to select the deployment engine (`terraform` or `direct`). The `DATABRICKS_BUNDLE_ENGINE` environment variable takes precedence over this setting. When the configured engine doesn't match existing deployment state, a warning is issued and the existing engine is used ([#4749](https://github.com/databricks/cli/pull/4749)).

### CLI

### Bundles
* Standardize `personal_schemas` enum across bundle templates ([#4401](https://github.com/databricks/cli/pull/4401))
* engine/direct: Fix permanent drift on experiment name field ([#4627](https://github.com/databricks/cli/pull/4627))
* engine/direct: Fix permissions state path to match input config schema ([#4703](https://github.com/databricks/cli/pull/4703))
* Add default project name and success message to default-scala template ([#4661](https://github.com/databricks/cli/pull/4661))

### Dependency updates

### API Changes
