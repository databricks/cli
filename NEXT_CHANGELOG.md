# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

* An explicitly selected profile (`--profile` or a bundle's `workspace.profile`) now takes precedence over auth environment variables (`DATABRICKS_HOST`, `DATABRICKS_TOKEN`, etc.) instead of being silently shadowed by them; env vars still fill auth fields the profile leaves empty ([#5096](https://github.com/databricks/cli/issues/5096)).
* `databricks apps run-local` now injects an `X-Forwarded-Access-Token` header minted from your CLI credentials, matching the deployed Apps OAuth2 proxy, so on-behalf-of (OBO) code paths can be exercised locally. When the current credentials are not OAuth-based (e.g. a PAT), the header is omitted with a warning and the proxy keeps working as before ([#5795](https://github.com/databricks/cli/pull/5795)).

### Bundles

* Fix permissions added to a job or pipeline by a Python (PyDABs) mutator failing to deploy with "must have exactly one owner"; the deploying identity is now set as owner, matching resources whose permissions are declared in YAML ([#5821](https://github.com/databricks/cli/pull/5821)).
* Remove duplicate enum values for jsonschema.json ([#5839](https://github.com/databricks/cli/pull/5839)).
* direct: volumes: support `volume_path` property ([#5550](https://github.com/databricks/cli/pull/5550)).
* direct: Fix deploy bug when a `postgres_projects`, `postgres_branches`, or `postgres_endpoints` field is set to its zero value (e.g. `enable_pg_native_login: false`, `replace_existing: false`) ([#5782](https://github.com/databricks/cli/pull/5782)).

### Dependency updates

### API Changes
