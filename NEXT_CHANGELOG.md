# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* Add `--concurrency` flag to `databricks sync` and `databricks bundle sync` to control the number of parallel requests to the workspace (default 20). Useful when uploading many medium-sized files where the previous fixed concurrency could trigger stream timeouts ([#5197](https://github.com/databricks/cli/pull/5197)).

### Bundles

### Dependency updates
