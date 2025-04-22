# NEXT CHANGELOG

## Release v0.249.0

### Notable Changes
* Added inline script execution support to bundle run. You can now run scripts in the same authentication context as a DAB using the databricks bundle run command. ([#2413](https://github.com/databricks/cli/pull/2413))

### Dependency updates

### CLI

### Bundles
* Correctly translate paths to local requirements.txt file in environment dependencies ([#2736](https://github.com/databricks/cli/pull/2736))
* Check for running resources with --fail-on-active-runs before any mutative operation during deploy ([#2743](https://github.com/databricks/cli/pull/2743))

### API Changes
