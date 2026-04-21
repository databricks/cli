# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.
* Deprecated `auth env`. The command is hidden from help listings and prints a deprecation warning to stderr; it will be removed in a future release.
* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.

### Bundles
* Remove `experimental-jobs-as-code` template, superseded by `pydabs` ([#4999](https://github.com/databricks/cli/pull/4999)).
* engine/direct: Added support for Vector Search Endpoints ([#4887](https://github.com/databricks/cli/pull/4887))
* engine/direct: Exclude deploy-only fields (e.g. `lifecycle`) from the Apps update mask so requests that change both `description` and `lifecycle.started` in the same deploy no longer fail with `INVALID_PARAMETER_VALUE`.

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.128.0 ([#4984](https://github.com/databricks/cli/pull/4984), [#5031](https://github.com/databricks/cli/pull/5031)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
