# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.

### Bundles
* Remove `experimental-jobs-as-code` template, superseded by `pydabs` ([#4999](https://github.com/databricks/cli/pull/4999)).
* engine/direct: Fix phantom diffs from `depends_on` reordering in job tasks (denik/jobs-depends-on)

* engine/direct: Added support for Vector Search Endpoints ([#4887](https://github.com/databricks/cli/pull/4887))

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
