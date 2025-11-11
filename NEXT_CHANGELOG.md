# NEXT CHANGELOG

## Release v0.277.0

### Notable Changes

### CLI
* Add `databricks query sql` for running SQL statements with NDJSON/CSV/table output, read-only safety gate, and Statement Execution polling.

### Dependency updates

### Bundles
* Add `default-minimal` template for users who want a clean slate without sample code ([#3885](https://github.com/databricks/cli/pull/3885))
* Add validation that served_models and served_entities are not used at the same time. Add client side translation logic. ([#3880](https://github.com/databricks/cli/pull/3880))

### API Changes
