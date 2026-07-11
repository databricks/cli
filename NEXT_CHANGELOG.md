# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* experimental `ssh connect`: bare `python`/`pip` in an interactive session now resolve to the environment interpreter (`$DATABRICKS_VIRTUAL_ENV`) instead of the system or cluster-libraries interpreter, so packages installed in the environment are importable without extra setup. The interactive shell is now non-login (`bash -i`) and the server seeds a `~/.bashrc` snippet that re-prepends the environment's bin directory to `PATH` ([#5888](https://github.com/databricks/cli/pull/5888)).

### Bundles

* `bundle generate job` can now download workspace files referenced by `spark_python_task`, rewriting them to a relative path like it already does for notebooks. This is opt-in via the `--download-spark-python-files` flag, because a Python file often imports sibling files that are not captured; Git-sourced files and cloud URIs are left untouched ([#5799](https://github.com/databricks/cli/pull/5799)).
* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).

### Dependency updates

### API Changes
