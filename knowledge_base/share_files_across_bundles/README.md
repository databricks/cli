# Share files across bundles

This example demonstrates how you can include files located outside the bundle root directory.
This is particularly useful when you use a single repository to host multiple bundles and
want to share common files across them (e.g. library code or configuration files).

Including files outside the bundle root directory is possible with the following configuration:

```yaml
sync:
  paths:
    - "../common"
    - "."
```

This configuration will sync the `common` directory located one level above the bundle root directory
and the bundle root directory itself. If the bundle root directory is named `my_bundle`, then the
file tree in the Databricks workspace will look like this:

```
common/
  common_file.txt
my_bundle/
  databricks.yml
  src/
    ...
```

## Prerequisites

* Databricks CLI v0.227.0 or above
* Serverless is enabled in your Databricks workspace

## Usage

Navigate to either one of the directories in this example:
* `bundle_with_shared_code`
* `bundle_with_shared_configuration`

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the job.

Run `databricks bundle run example_job_with_notebook` to run the notebook task.

Run `databricks bundle run example_job_with_python_file` to run the Python task.

In case of the notebook, navigate to the run page in the Databricks workspace to see the output.

In case of the Python file, the output is shown in your terminal.
