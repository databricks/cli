# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).
* Fix intermittent crashes when processing pages from API calls ([#5815](https://github.com/databricks/cli/pull/5815)).

### Bundles

* direct: add basic version of job_runs resource (experimental) ([#5603](https://github.com/databricks/cli/pull/5603)).
* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
* direct: volumes: support `volume_path` property ([#5550](https://github.com/databricks/cli/pull/5550)).
* direct: Fix deploy bug when a `postgres_projects`, `postgres_branches`, or `postgres_endpoints` field is set to its zero value (e.g. `enable_pg_native_login: false`, `replace_existing: false`) ([#5782](https://github.com/databricks/cli/pull/5782)).
* `bundle run --only` help now documents the `+` modifier syntax: prefix a task key with `+` to also run its upstream tasks, or suffix it with `+` for downstream tasks ([#5760](https://github.com/databricks/cli/pull/5760)).
* direct: Recognize UC-managed catalog and schema property defaults to avoid unnecessary drift ([#5865](https://github.com/databricks/cli/pull/5865) & [#5870](https://github.com/databricks/cli/pull/5870)).
* Fix `bundle deploy --select <resource>` skipping the resource's grants and permissions; they are now applied as part of the selected resource ([#5852](https://github.com/databricks/cli/pull/5852)).
* Support `purge_on_delete: true` on `postgres_branches` so bundles can hard-delete a Lakebase branch on destroy (skipping the soft-delete retention window) ([#5801](https://github.com/databricks/cli/pull/5801)).
* Support `replace_existing: true` on `postgres_databases` and `postgres_roles` so bundles can take over a database or role that already exists on a Lakebase branch instead of failing with `ALREADY_EXISTS` ([#5803](https://github.com/databricks/cli/pull/5803)).

### Dependency updates

* Bump databricks-sdk-go to v0.154.0 ([#5855](https://github.com/databricks/cli/pull/5855)).
* Bump terraform-provider to 1.121.0 ([#5857](https://github.com/databricks/cli/pull/5857)).
* Bump OpenTelemetry dependencies to v1.44.0 to address [CVE-2026-41178](https://github.com/advisories/GHSA-5wrp-cwcj-q835) ([#5873](https://github.com/databricks/cli/pull/5873)).

### API Changes
