# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI
* Add `databricks dbconnect sync` to provision a local Python environment (Python version, `databricks-connect` pin, and dependency constraints) matched to the selected Databricks compute target. A project with no `pyproject.toml` is initialized from scratch; an existing one is merged in place. Supports `--cluster`/`--serverless`/`--job` target selection, `--constraints-only` (skip `databricks-connect`), `--check` (dry run), and `--output json` for the VS Code extension.

### Bundles

### Dependency updates

### API Changes
