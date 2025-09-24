# NEXT CHANGELOG

## Release v0.270.0

### Notable Changes
* Add 'databricks bundle plan' command. This command shows the deployment plan for the current bundle configuration without making any changes. ([#3530](https://github.com/databricks/cli/pull/3530)

### CLI

### Dependency updates

### Bundles
* Add 'databricks bundle plan' command ([#3530](https://github.com/databricks/cli/pull/3530)
* Add new Lakeflow Pipelines support for bundle generate ([#3568](https://github.com/databricks/cli/pull/3568))
* Fix bundle deploy to not update permissions or grants for unbound resources ([#3642](https://github.com/databricks/cli/pull/3642))
* Introduce new bundle variable: `${workspace.current_user.domain_friendly_name}` ([#3623](https://github.com/databricks/cli/pull/3623))
* Improve the output of bundle run when bundle is not deployed ([#3652](https://github.com/databricks/cli/pull/3652))

### API Changes
