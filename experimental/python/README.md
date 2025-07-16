# databricks-bundles

Python for Databricks Asset Bundles extends [Databricks Asset Bundles](https://docs.databricks.com/aws/en/dev-tools/bundles/) so that you can:
- Define jobs and pipelines as Python code. These jobs can coexist with jobs defined in YAML.
- Dynamically create jobs and pipelines using metadata.
- Modify jobs and pipelines defined in YAML or Python during bundle deployment.

Documentation is available at https://docs.databricks.com/dev-tools/cli/databricks-cli.html.

Reference documentation is available at https://databricks.github.io/cli/experimental/python/

## Getting started

To use `databricks-bundles`, you must first:

1. Install the [Databricks CLI](https://github.com/databricks/cli), version 0.260.0 or above
2. Authenticate to your Databricks workspace if you have not done so already:

   ```bash
   databricks configure
   ```
3. To create a new project, initialize a bundle using the `experimental-jobs-as-code` template:

  ```bash
  databricks bundle init experimental-jobs-as-code
  ```

## Privacy Notice
Databricks CLI use is subject to the [Databricks License](https://github.com/databricks/cli/blob/main/LICENSE) and [Databricks Privacy Notice](https://www.databricks.com/legal/privacynotice), including any Usage Data provisions.
