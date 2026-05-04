# NEXT CHANGELOG

## Release v0.300.0

### CLI

* `databricks api` now works against unified hosts. Adds `--account` to scope a call to the account API, `--workspace-id` to override the workspace routing identifier per call, and `{account_id}` substitution from the active profile's `account_id`. A `?o=<workspace-id>` query parameter on the path (the SPOG URL convention used by the Databricks UI) is also recognized as a per-call workspace override, so URLs pasted from the browser route correctly.

### Bundles

### Dependency updates
