# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).

### Bundles

* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
* Expose a computed, read-only `volume_path` on `resources.volumes.*` so configs can reference a volume's Unity Catalog path via `${resources.volumes.<key>.volume_path}` instead of hardcoding `/Volumes/<catalog>/<schema>/<name>` ([#5550](https://github.com/databricks/cli/pull/5550)).
  * `volume_path` is derived purely from the volume's `catalog_name`, `schema_name`, and `name`, so the reference is resolved early (at initialize) and inlined into the referring field. Referencing `volume_path` therefore does not make the referring resource depend on the volume during deploy; if `catalog_name`/`schema_name`/`name` themselves reference other resources, the referrer depends on those resources instead.
  * Supported on the direct deployment engine (`DATABRICKS_BUNDLE_ENGINE=direct`). On the Terraform engine `volume_path` is dropped before apply, and a `volume_path` whose components embed a value only known after deploy (for example `${resources.jobs.<key>.creator_user_name}`) is not supported.

### Dependency updates

### API Changes
