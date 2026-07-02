# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* Wait for the SSH server health check to pass for the full startup timeout (10 minutes, or 45 minutes with `--accelerator`) instead of a fixed 60 seconds, so `databricks ssh connect` no longer fails with a driver-proxy 503 while a custom `--base-environment` finishes installing ([#5807](https://github.com/databricks/cli/pull/5807)).

### Bundles

### Dependency updates

### API Changes
