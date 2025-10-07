# NEXT CHANGELOG

## Release v0.272.0

### Notable Changes

### CLI

### Dependency updates

### Bundles
* Updated the internal lakeflow-pipelines template to use an "src" layout ([#3671](https://github.com/databricks/cli/pull/3671)).
* Added support for a "template_dir" option in the databricks_template_schema.json format. ([#3686](https://github.com/databricks/cli/pull/3686)).
* Remove resources.apps.config section ([#3680](https://github.com/databricks/cli/pull/3680))
* Prompt for serverless compute in `dbt-sql` template (defaults to `yes`) ([#3668](https://github.com/databricks/cli/pull/3668))
* Fix processing short pip flags in environment dependencies ([#3708](https://github.com/databricks/cli/pull/3708))
* Add support for referencing local files in -e pip flag for environment dependencies ([#3708](https://github.com/databricks/cli/pull/3708))

### API Changes
