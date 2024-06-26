# Job with multiple wheels

This example demonstrates how to define and use a job with multiple wheel dependencies in a Databricks Asset Bundle.

One of the wheel files depends on the other. It is important to specify the order of the wheels in the job such that
the dependent wheel is installed first, since it won't be available in a public registry.

## Prerequisites

* Databricks CLI v0.222.0 or above

## Usage

Update the `host` field under `workspace` in `databricks.yml` to the Databricks workspace you wish to deploy to.

Run `databricks bundle deploy` to deploy the job.

Run `databricks bundle run example_job` to run the job.

Example output:

```
$ databricks bundle run example_job
Run URL: https://...

2024-06-26 10:18:36 "[dev pieter_noordhuis] Example with multiple wheels" TERMINATED SUCCESS
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
