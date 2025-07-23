# NEXT CHANGELOG

## Release v0.261.0

### Notable Changes
The following CLI commands now have a required positional argument "UPDATE_MASK":
* `alerts update` - Update an alert
* `alerts-v2 update-alert` - Update an alert (v2)
* `clusters update` - Update a cluster
* `database update-database-instance` - Update a database instance
* `external-lineage update-external-lineage-relationship` - Update an external lineage relationship
* `external-metadata update-external-metadata` - Update external metadata
* `feature-store update-online-store` - Update an online store
* `network-connectivity update-private-endpoint-rule` - Update a private endpoint rule
* `queries update` - Update a query
* `query-visualizations update` - Update a query visualization

### Dependency updates

### CLI
* Add required query parameters as positional arguments in CLI commands ([#3289](https://github.com/databricks/cli/pull/3289))

### Bundles
* Fixed an issue where `allow_duplicate_names` field on the pipeline definition was ignored by the bundle ([#3274](https://github.com/databricks/cli/pull/3274))

### API Changes
