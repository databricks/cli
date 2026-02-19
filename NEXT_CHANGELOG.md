# NEXT CHANGELOG

## Release v0.289.0

### CLI
* Make auth profiles respect DATABRICKS_CLI_PATH env var (([#4467](https://github.com/databricks/cli/pull/4467)))
* Fix arrow key navigation in prompts on Windows (([#4501](https://github.com/databricks/cli/pull/4501)))

### Bundles
* Log artifact build output in debug mode ([#4208](https://github.com/databricks/cli/pull/4208))
* Fix bundle init not working in Azure Government ([#4286](https://github.com/databricks/cli/pull/4286))
* Allow single and double quotes in environment dependencies (([#4511](https://github.com/databricks/cli/pull/4511)))
* Use purge option when deleting alerts (([#4505](https://github.com/databricks/cli/pull/4505)))
* engine/direct: Replace server_side_default with more precise backend_default rule in bundle plan ([#4490](https://github.com/databricks/cli/pull/4490))
* engine/direct: Extend pipelines recreate_on_changes configuration (([#4499](https://github.com/databricks/cli/pull/4499)))
* engine/direct: Added support for UC external locations (direct only) ([#4484](https://github.com/databricks/cli/pull/4484))

### Dependency updates
* Upgrade Go SDK to v0.106.0 (([#4486](https://github.com/databricks/cli/pull/4486)))
* Upgrade Terraform provider to v1.106.0 (([#4542](https://github.com/databricks/cli/pull/4542)))
