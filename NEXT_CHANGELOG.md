# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Deprecated `auth env`. The command is now hidden from help listings and prints a deprecation warning to stderr; it will be removed in a future release. It has also been refactored to use the CLI's standard auth resolution and added `--output text` support. Breaking: removed the command-specific `--host`/`--profile` flags (use the inherited ones) and only the primary env var per attribute is emitted ([#4904](https://github.com/databricks/cli/pull/4904)).
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.

### Bundles
* Remove `experimental-jobs-as-code` template, superseded by `pydabs` ([#4999](https://github.com/databricks/cli/pull/4999)).
* engine/direct: Added support for Vector Search Endpoints ([#4887](https://github.com/databricks/cli/pull/4887))

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
