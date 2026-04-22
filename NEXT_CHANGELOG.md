# NEXT CHANGELOG

## Release v0.298.0

### CLI
* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)). On `jobs list` and `jobs list-runs` the former API page-size flag was renamed to `--page-size` (hidden) to avoid collision.
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.
* Cache `/.well-known/databricks-config` lookups under `~/.cache/databricks/<version>/host-metadata/` so repeat CLI invocations against the same host skip the ~700ms discovery round trip.
* Deprecated `auth env`. The command is hidden from help listings and prints a deprecation warning to stderr; it will be removed in a future release.

### Bundles
* Remove `experimental-jobs-as-code` template, superseded by `pydabs` ([#4999](https://github.com/databricks/cli/pull/4999)).
* Prompt before destroying or recreating Lakebase resources (database instances, synced database tables, postgres projects and branches) ([#5052](https://github.com/databricks/cli/pull/5052)).
* Treat deleted resources as not running in the `fail-on-active-runs` check ([#5044](https://github.com/databricks/cli/pull/5044)).
* engine/direct: Added support for Vector Search Endpoints ([#4887](https://github.com/databricks/cli/pull/4887)).
* engine/direct: Exclude deploy-only fields (e.g. `lifecycle`) from the Apps update mask so requests that change both `description` and `lifecycle.started` in the same deploy no longer fail with `INVALID_PARAMETER_VALUE` ([#5042](https://github.com/databricks/cli/pull/5042), [#5051](https://github.com/databricks/cli/pull/5051)).
* engine/direct: Fix phantom diffs from `depends_on` reordering in job tasks ([#4990](https://github.com/databricks/cli/pull/4990)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.128.0 ([#4984](https://github.com/databricks/cli/pull/4984), [#5031](https://github.com/databricks/cli/pull/5031)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004)).
