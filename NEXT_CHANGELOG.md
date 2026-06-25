# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

### Bundles

 * direct: Cluster resize now falls back to regular update if resize fails due to `INVALID_STATE` ([#5716](https://github.com/databricks/cli/pull/5716)).
* Allow bundles with `apps` resources to have a top-level `run_as` identity configured. Apps do not support `run_as` via the API and are simply skipped; other resources (jobs, pipelines, etc.) continue to have `run_as` applied as before.

### Dependency updates

### API Changes
