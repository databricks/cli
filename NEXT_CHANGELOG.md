# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* Promote the aitools skills-management surface (`install`, `update`, `uninstall`, `list`, `version`) from `databricks experimental aitools` to top-level `databricks aitools`. The old paths under `databricks experimental aitools` continue to work as silent backward-compat aliases. The `tools` subtree (`query`, `discover-schema`, `get-default-warehouse`, `statement …`) and the `skills` alias group remain under `databricks experimental aitools`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))

### Dependency updates
