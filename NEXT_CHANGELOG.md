# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.
* Stop persisting `experimental_is_unified_host` to new profiles and ignore the `DATABRICKS_EXPERIMENTAL_IS_UNIFIED_HOST` env var. Unified hosts are now detected automatically from `/.well-known/databricks-config`. Existing profiles with the key set continue to work via a legacy fallback; `--experimental-is-unified-host` is deprecated but still honored as a routing fallback for this release.

### Bundles

### Dependency updates
