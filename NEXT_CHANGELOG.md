# NEXT CHANGELOG

## Release v0.300.0

### CLI

* Promote the aitools skills-management surface (`install`, `update`, `uninstall`, `list`, `version`) from `databricks experimental aitools` to top-level `databricks aitools`. The old paths under `databricks experimental aitools` continue to work as silent backward-compat aliases. The `tools` subtree (`query`, `discover-schema`, `get-default-warehouse`, `statement …`) and the `skills` alias group remain under `databricks experimental aitools`.
* `databricks api` now works against unified hosts. Adds `--account` to scope a call to the account API and `--workspace-id` to override the workspace routing identifier per call. A `?o=<workspace-id>` query parameter on the path (the SPOG URL convention used by the Databricks UI) is also recognized as a per-call workspace override, so URLs pasted from the browser route correctly.

### Bundles

### Dependency updates
