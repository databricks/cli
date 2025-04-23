# NEXT CHANGELOG

## Release v0.249.0

### Notable Changes

### Dependency updates

### CLI
* Added `exclude-from` and `include-from` flags support to sync command ([#2660](https://github.com/databricks/cli/pull/2660))

### Bundles
* Correctly translate paths to local requirements.txt file in environment dependencies ([#2736](https://github.com/databricks/cli/pull/2736))
* Check for running resources with --fail-on-active-runs before any mutative operation during deploy ([#2743](https://github.com/databricks/cli/pull/2743))
* Raise an error when Volumes are used for paths other than artifacts ([#2754](https://github.com/databricks/cli/pull/2754))

### API Changes
