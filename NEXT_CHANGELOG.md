# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).

### Bundles

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).

### API Changes

* Added `apply-environment` command to pipelines service.
* Added `ManagedEncryptionSettings` field to catalog resources.
* Added `EffectiveFileEventQueue` field to external location resources.
* Added `DefaultBranch` field to Postgres project resources.

OpenAPI spec updated via [databricks-sdk-go v0.127.0](https://github.com/databricks/databricks-sdk-go/releases/tag/v0.127.0).
