# NEXT CHANGELOG

## Release v0.296.0

### Notable Changes

### CLI
* Add `--force-refresh` flag to `databricks auth token` to force a token refresh even when the cached token is still valid ([#4767](https://github.com/databricks/cli/pull/4767)).

### Bundles
* Added support for lifecycle.started option for apps ([#4672](https://github.com/databricks/cli/pull/4672))
* engine/direct: Fix drift in grants resource due to privilege reordering ([#4794](https://github.com/databricks/cli/pull/4794))
* engine/direct: Fix 400 error when deploying grants with ALL_PRIVILEGES ([#4801](https://github.com/databricks/cli/pull/4801))
* Deduplicate grant entries with duplicate principals or privileges during initialization ([#4801](https://github.com/databricks/cli/pull/4801))
* engine/direct: Fix unwanted recreation of secret scopes when scope_backend_type is not set ([#4834](https://github.com/databricks/cli/pull/4834))
* engine/direct: Fix bind and unbind for non-Terraform resources ([#4850](https://github.com/databricks/cli/pull/4850))
* engine/direct: Fix deploying removed principals ([#4824](https://github.com/databricks/cli/pull/4824))

### Dependency updates

### API Changes
