# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* experimental `ssh connect`: bare `python`/`pip` in an interactive session now resolve to the environment interpreter (`$DATABRICKS_VIRTUAL_ENV`) instead of the system or cluster-libraries interpreter, so packages installed in the environment are importable without extra setup. The interactive shell is now non-login (`bash -i`) and the server seeds a `~/.bashrc` snippet that re-prepends the environment's bin directory to `PATH` ([#5888](https://github.com/databricks/cli/pull/5888)).

### Bundles

* Added an `env:` section to `scripts.<name>` for declaring environment variables that may reference `${bundle.*}`, `${workspace.*}`, and `${var.*}` ([#4179](https://github.com/databricks/cli/issues/4179)).
* `bundle validate` now fails early with a clear error when a `sql_warehouse` is missing a `name`, a grant is missing a `principal`, or a catalog/schema `custom_max_retention_hours` is outside the allowed range (0 or 168-720 hours), instead of passing validation and failing later at deploy.

### Dependency updates

### API Changes
