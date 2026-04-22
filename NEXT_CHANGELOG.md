# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.
* Stop persisting `experimental_is_unified_host` to new profiles and ignore the `DATABRICKS_EXPERIMENTAL_IS_UNIFIED_HOST` env var. Unified hosts are now detected automatically from `/.well-known/databricks-config`. Existing profiles with the key set continue to work via a legacy fallback; `--experimental-is-unified-host` is deprecated but still honored as a routing fallback for this release.

### Bundles

### Dependency updates

* Added `github.com/zalando/go-keyring` as a new dependency (dormant until a later release enables experimental secure-storage for OAuth tokens).
