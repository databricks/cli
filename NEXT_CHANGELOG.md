# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).

### Bundles
* Fix nil pointer dereference in `WaitForDeploymentToComplete` when app deployment status is nil (denik/random-bugfixes-5)

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
