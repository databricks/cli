# NEXT CHANGELOG

## Release v0.255.0

### Notable Changes

* Fix `databricks auth login` to tolerate URLs copied from the browser ([#3001](https://github.com/databricks/cli/pull/3001)).

### Dependency updates

### CLI
* Use OS aware runner instead of bash for run-local command ([#2996](https://github.com/databricks/cli/pull/2996))

### Bundles
* Fix "bundle summary -o json" to render null values properly ([#2990](https://github.com/databricks/cli/pull/2990))
* Fix dashboard generation for already imported dashboard ([#3016](https://github.com/databricks/cli/pull/3016))
* Fixed null pointer de-reference if artifacts missing fields ([#3022](https://github.com/databricks/cli/pull/3022))
* Update bundle templates to also include `resources/*/*.yml` ([#3024](https://github.com/databricks/cli/pull/3024))
* Apply YAML formatter on default-python and dbt-sql templates ([#3026](https://github.com/databricks/cli/pull/3026))

### API Changes
