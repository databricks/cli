# NEXT CHANGELOG

## Release v0.300.0

### Notable Changes

### CLI

* `[__settings__].default_profile` is now honored by `databricks api`, `databricks auth token`, and bundle commands when no `--profile` flag and no `DATABRICKS_CONFIG_PROFILE` env var is set. For bundle commands, `default_profile` only applies when the bundle does not pin its own `workspace.host`.

### Bundles
* Make sure warnings asking for approval are understood by agents ([#5239](https://github.com/databricks/cli/pull/5239))

### Dependency updates
