# NEXT CHANGELOG

## Release v0.255.0

### Notable Changes

* Fix `databricks auth login` to tolerate URLs copied from the browser ([#3001](https://github.com/databricks/cli/pull/3001)).

### Dependency updates

### CLI
* Use OS aware runner instead of bash for run-local command ([#2996](https://github.com/databricks/cli/pull/2996))

### Bundles
* Fix "bundle summary -o json" to render null values properly ([#2990](https://github.com/databricks/cli/pull/2990))
* When glob for wheels is used, like "\*.whl", it will filter out different version of the same package and will only take the most recent version. ([#2982](https://github.com/databricks/cli/pull/2982))
* When building Python artifacts as part of "bundle deploy" we no longer delete `dist`, `build`, `*egg-info` and `__pycache__` directories. ([#2982](https://github.com/databricks/cli/pull/2982))

### API Changes
