# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Refactored `auth env` to use the CLI's standard auth resolution and added `--output text` support. The command is now hidden from help listings. Breaking: removed the command-specific `--host`/`--profile` flags (use the inherited ones) and only the primary env var per attribute is emitted ([#4904](https://github.com/databricks/cli/pull/4904)).

### Bundles

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
