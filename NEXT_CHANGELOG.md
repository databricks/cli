# NEXT CHANGELOG

## Release v1.4.0

### Notable Changes

### CLI
* Improved error messages for `ssh connect`: when an SSH connection attempt fails, the client now fetches and prints the server's recent error logs ([#5555](https://github.com/databricks/cli/pull/5555)).
* Increase the SSH server startup timeout from 10 to 45 minutes when a GPU accelerator is requested via `databricks ssh connect --accelerator` ([#5569](https://github.com/databricks/cli/pull/5569)).
* Fix authentication falling back to the default profile in `.databrickscfg` when a host is already configured via the environment (e.g. `DATABRICKS_HOST` with `DATABRICKS_TOKEN`) ([#5616](https://github.com/databricks/cli/pull/5616)).


### Bundles
* Remove API enum values and types that are still in development from the `databricks-bundles` Python package; these were never accepted by the backend ([#5484](https://github.com/databricks/cli/pull/5484)).
* direct: Fix resolving a resource reference that is used more than once within the same field ([#5558](https://github.com/databricks/cli/pull/5558)).
* Bundle variable references now accept Unicode letters in path segments (e.g. `${var.变量}`). ([#5532](https://github.com/databricks/cli/pull/5532))
* Ignore remote changes for vector search direct_access_index_spec.schema_json to prevent drift when the backend normalizes the schema ([#5481](https://github.com/databricks/cli/pull/5481)).
* Remove hidden, never-functional `--existing-dashboard-id`, `--existing-dashboard-path`, `--existing-alert-id`, and `--existing-genie-space-id` alias flags from `bundle generate`; use the documented `--existing-id` / `--existing-path` flags instead ([#5591](https://github.com/databricks/cli/pull/5591)).
* engine/direct: Fix WAL corruption after two consecutive failed deploys ([#5606](https://github.com/databricks/cli/pull/5606)).
* engine/direct: Don't open the deployment state WAL when a deploy's plan fails ([#5607](https://github.com/databricks/cli/pull/5607)).
* Ignore unity catalog managed schema property defaults to avoid unnecessary drift ([#5195](https://github.com/databricks/cli/pull/5195)).
* Set the default `data_security_mode` to `DATA_SECURITY_MODE_AUTO` in bundle templates ([#5452](https://github.com/databricks/cli/pull/5452)).
* Add Postgres role as a bundle resource ([#5467](https://github.com/databricks/cli/pull/5467)).
* Add support for `postgres_databases` bundle resource.

### Dependency updates

### API Changes
