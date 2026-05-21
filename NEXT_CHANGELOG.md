# NEXT CHANGELOG

## Release v1.1.0

### Notable Changes

### CLI
* Ctrl+C in an interactive prompt now prints `cancelled` and exits 130 instead of `Error: user aborted` / `Error: ^C` and exit 1. Applies to all `huh`-based prompts (e.g. `databricks aitools`, `databricks apps init`) and the bubbletea-based prompts in `libs/cmdio`.

### Bundles
* The error reported when a direct-only resource (catalogs, external locations, vector search endpoints) is used with the terraform engine now also suggests setting `bundle.engine: direct` in `databricks.yml`, in addition to the `DATABRICKS_BUNDLE_ENGINE` environment variable ([#5295](https://github.com/databricks/cli/pull/5295)).
