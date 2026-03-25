# NEXT CHANGELOG

## Release v0.296.0

### Notable Changes

### CLI

### Bundles
* engine/direct: Fix drift in grants resource due to privilege reordering ([#4794](https://github.com/databricks/cli/pull/4794))
* engine/direct: Fix 400 error when deploying grants with ALL_PRIVILEGES ([#4801](https://github.com/databricks/cli/pull/4801))
* Deduplicate grant entries with duplicate principals or privileges during initialization ([#4801](https://github.com/databricks/cli/pull/4801))
* engine/direct: Fix unwanted recreation of secret scopes when scope_backend_type is not set ([#4834](https://github.com/databricks/cli/pull/4834))

* **Breaking**: Nested variable references like `${var.foo_${var.tail}}` are now rejected with a warning and left unresolved. Previously the regex-based parser matched only the innermost `${var.tail}` by coincidence, which silently produced incorrect results. If you rely on dynamic variable name construction, use separate variables or target overrides instead ([#4747](https://github.com/databricks/cli/pull/4747)).

### Dependency updates

### API Changes
