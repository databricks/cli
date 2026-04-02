# NEXT CHANGELOG

## Release v0.296.0

### Notable Changes

### CLI
* Auth commands now error when --profile and --host conflict ([#4841](https://github.com/databricks/cli/pull/4841))
* Add `--force-refresh` flag to `databricks auth token` to force a token refresh even when the cached token is still valid ([#4767](https://github.com/databricks/cli/pull/4767)).

### Bundles
* engine/direct: Fix drift in grants resource due to privilege reordering ([#4794](https://github.com/databricks/cli/pull/4794))
* engine/direct: Fix 400 error when deploying grants with ALL_PRIVILEGES ([#4801](https://github.com/databricks/cli/pull/4801))
* Deduplicate grant entries with duplicate principals or privileges during initialization ([#4801](https://github.com/databricks/cli/pull/4801))
* engine/direct: Fix unwanted recreation of secret scopes when scope_backend_type is not set ([#4834](https://github.com/databricks/cli/pull/4834))
* engine/direct: Fix bind and unbind for non-Terraform resources ([#4850](https://github.com/databricks/cli/pull/4850))
* Fix `bundle deployment bind` to always pull remote state before modifying (denik/investigate-dir)
* engine/direct: Fix deploying removed principals ([#4824](https://github.com/databricks/cli/pull/4824))
* engine/direct: Fix secret scope permissions migration from Terraform to Direct engine ([#4866](https://github.com/databricks/cli/pull/4866))

### Dependency updates

### API Changes
