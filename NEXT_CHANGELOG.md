# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Moved file-based OAuth token cache management from the SDK to the CLI. No user-visible change; part of a three-PR sequence that makes the CLI the sole owner of its token cache.

### Bundles

* engine/direct: Added support for Vector Search Endpoints ([#4887](https://github.com/databricks/cli/pull/4887))
* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.

### Dependency updates

* Added `github.com/zalando/go-keyring` as a new dependency (dormant until a later release enables experimental secure-storage for OAuth tokens).
