# NEXT CHANGELOG

## Release v0.300.0

### CLI

### Bundles

* Propagate authentication environment (including `DATABRICKS_CONFIG_PROFILE`) to the `experimental.python` subprocess so bundle validate/deploy no longer fails with a multi-profile host ambiguity error when several profiles in `~/.databrickscfg` share the same host.

### Dependency updates
