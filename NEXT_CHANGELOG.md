# NEXT CHANGELOG

## Release v0.261.0

### Notable Changes
The following CLI commands now have additional required positional arguments:
* `alerts-v2 update-alert ID UPDATE_MASK` - Update an alert (v2)
* `database update-database-instance NAME UPDATE_MASK` - Update a database instance
* `external-lineage create-external-lineage-relationship SOURCE TARGET` - Create an external lineage relationship
* `external-lineage update-external-lineage-relationship UPDATE_MASK SOURCE TARGET` - Update an external lineage relationship
* `external-metadata update-external-metadata NAME UPDATE_MASK SYSTEM_TYPE ENTITY_TYPE` - Update external metadata
* `feature-store update-online-store NAME UPDATE_MASK CAPACITY` - Update an online store
* `lakeview create-schedule DASHBOARD_ID CRON_SCHEDULE` - Create a schedule
* `lakeview create-subscription DASHBOARD_ID SCHEDULE_ID SUBSCRIBER` - Create a subscription
* `lakeview update-schedule DASHBOARD_ID SCHEDULE_ID CRON_SCHEDULE` - Update a schedule
* `network-connectivity update-private-endpoint-rule NETWORK_CONNECTIVITY_CONFIG_ID PRIVATE_ENDPOINT_RULE_ID UPDATE_MASK` - Update a private endpoint rule

### Dependency updates

### CLI
* Add required query parameters as positional arguments in CLI commands ([#3289](https://github.com/databricks/cli/pull/3289))

### Bundles
* Fixed an issue where `allow_duplicate_names` field on the pipeline definition was ignored by the bundle ([#3274](https://github.com/databricks/cli/pull/3274))
* Add warning for when required bundle fields are not set ([#3044](https://github.com/databricks/cli/pull/3044))

### API Changes
