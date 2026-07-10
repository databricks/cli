# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* experimental `ssh connect`: bare `python`/`pip` in an interactive session now resolve to the environment interpreter (`$DATABRICKS_VIRTUAL_ENV`) instead of the system or cluster-libraries interpreter, so packages installed in the environment are importable without extra setup. The interactive shell is now non-login (`bash -i`) and the server seeds a `~/.bashrc` snippet that re-prepends the environment's bin directory to `PATH` ([#5888](https://github.com/databricks/cli/pull/5888)).

### Bundles

### Dependency updates

### API Changes
