# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).
* `databricks ssh connect` now retries the SSH tunnel binary upload on transient workspace-files errors (a cold `get-status` request timeout or a 5xx), instead of aborting the whole connect on a single blip ([#5836](https://github.com/databricks/cli/pull/5836)).

### Bundles

* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
* direct: volumes: support `volume_path` property ([#5550](https://github.com/databricks/cli/pull/5550)).

### Dependency updates

### API Changes
