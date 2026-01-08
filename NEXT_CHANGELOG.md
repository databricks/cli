# NEXT CHANGELOG

## Release v0.282.0

### Notable Changes

### CLI
* Skip non-exportable objects (e.g., `MLFLOW_EXPERIMENT`) during `workspace export-dir` instead of failing ([#4081](https://github.com/databricks/cli/issues/4081))

* Improve performance of `databricks fs cp` command by parallelizing file uploads when
  copying directories with the `--recursive` flag.

### Bundles

### Dependency updates

### API Changes
