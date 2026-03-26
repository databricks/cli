# NEXT CHANGELOG

## Release v0.296.0

### Notable Changes

### CLI

### Bundles
* engine/direct: Fix drift in grants resource due to privilege reordering ([#4794](https://github.com/databricks/cli/pull/4794))
* engine/direct: Fix 400 error when deploying grants with ALL_PRIVILEGES ([#4801](https://github.com/databricks/cli/pull/4801))
* Deduplicate grant entries with duplicate principals or privileges during initialization ([#4801](https://github.com/databricks/cli/pull/4801))
* engine/direct: Fix unwanted recreation of secret scopes when scope_backend_type is not set ([#4834](https://github.com/databricks/cli/pull/4834))
* engine/direct: Fix bind and unbind for non-Terraform resources ([#4850](https://github.com/databricks/cli/pull/4850))

### Dependency updates

### API Changes
