# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.
* Remove the `--experimental-is-unified-host` flag and stop reading `experimental_is_unified_host` from `.databrickscfg` profiles and the `DATABRICKS_EXPERIMENTAL_IS_UNIFIED_HOST` env var. Unified hosts are now detected exclusively from `/.well-known/databricks-config` discovery. The `experimental_is_unified_host` field is retained as a no-op in `databricks.yml` for schema compatibility.

### Bundles

### Dependency updates

* Added `github.com/zalando/go-keyring` as a new dependency (dormant until a later release enables experimental secure-storage for OAuth tokens).
