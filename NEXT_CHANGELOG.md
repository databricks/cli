# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).

### Bundles

* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
* Expose a computed, read-only `volume_path` on `resources.volumes.*` so configs can reference a volume's Unity Catalog path via `${resources.volumes.<key>.volume_path}` instead of hardcoding `/Volumes/<catalog>/<schema>/<name>` ([#5550](https://github.com/databricks/cli/pull/5550)). Derived from `catalog_name`/`schema_name`/`name` and resolved at initialize, so the reference depends on those underlying resources rather than the volume itself. The field is computed by the CLI and never sent to the API (dropped before Terraform apply). References resolve on both engines when the path components are known at initialize; components that are only known after deploy (for example a remote `creator_user_name`) are supported on the direct engine (the default) but not on Terraform.

### Dependency updates

### API Changes
