# NEXT CHANGELOG

## Release v0.281.0

### Notable Changes

### CLI
* Add `databricks query sql` for running SQL statements with NDJSON/CSV/table output, read-only safety gate, and Statement Execution polling.

### Bundles
* engine/direct: Fix dependency-ordered deletion by persisting depends_on in state ([#4105](https://github.com/databricks/cli/pull/4105))
* Pass SYSTEM_ACCESSTOKEN from env to the Terraform provider ([#4135](https://github.com/databricks/cli/pull/4135)

### Dependency updates

### API Changes
