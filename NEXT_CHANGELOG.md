# NEXT CHANGELOG

## Release v0.271.0

### Notable Changes

### CLI

### Dependency updates

### Bundles

* Updated the internal lakeflow-pipelines template to use an "src" layout ([#3671](https://github.com/databricks/cli/pull/3671)).
* Added support for a "template_dir" option in the databricks_template_schema.json format. ([#3671](https://github.com/databricks/cli/pull/3671)).
* Remove resources.apps.config section ([#3680](https://github.com/databricks/cli/pull/3680))
* Prompt for serverless compute in `dbt-sql` template (defaults to `yes`) ([#3668](https://github.com/databricks/cli/pull/3668))

### API Changes
* Added `databricks account account-groups-v2` command group.
* Added `databricks account account-iam-v2` command group.
* Added `databricks feature-engineering` command group.
* Added `databricks shares list-shares` command.
