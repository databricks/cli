# Serverless job

This example demonstrates how to define and use a serverless job in a Databricks Asset Bundle.

For more information, please refer to the [documentation](https://docs.databricks.com/en/workflows/jobs/how-to/use-bundles-with-jobs.html#configure-a-job-that-uses-serverless-compute).

## Prerequisites

* Databricks CLI v0.218.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the job.

Run `databricks bundle run serverless_job` to run the job.

Example output:

```
$ databricks bundle run serverless_job
Run URL: https://...

2024-06-17 09:58:06 "[dev pieter_noordhuis] Example serverless job" TERMINATED SUCCESS
  _____________
| Hello, world! |
  =============
             \
              \
                ^__^
                (oo)\_______
                (__)\       )\/\
                    ||----w |
                    ||     ||
```
