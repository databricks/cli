# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* experimental `ssh connect`: bare `python`/`pip` in an interactive session now resolve to the environment interpreter (`$DATABRICKS_VIRTUAL_ENV`) instead of the system or cluster-libraries interpreter, so packages installed in the environment are importable without extra setup. The interactive shell is now non-login (`bash -i`) and the server seeds a `~/.bashrc` snippet that re-prepends the environment's bin directory to `PATH` ([#5888](https://github.com/databricks/cli/pull/5888)).

### Bundles
* engine/direct: Add declarative `bind` blocks under a target to bring existing workspace resources under bundle management at deploy time, with `bind` and `bind_and_update` actions surfaced in `bundle plan` output ([#4630](https://github.com/databricks/cli/pull/4630)).

* Added an `env:` section to `scripts.<name>` for declaring environment variables that may reference `${bundle.*}`, `${workspace.*}`, and `${var.*}` ([#4179](https://github.com/databricks/cli/issues/4179)).

### Dependency updates

### API Changes
