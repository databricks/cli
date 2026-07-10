# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

### Bundles

* `bundle generate job` can now download workspace files referenced by `spark_python_task`, rewriting them to a relative path like it already does for notebooks. This is opt-in via the `--download-spark-python-files` flag, because a Python file often imports sibling files that are not captured; Git-sourced files and cloud URIs are left untouched ([#5799](https://github.com/databricks/cli/pull/5799)).
* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).

### Dependency updates

### API Changes
