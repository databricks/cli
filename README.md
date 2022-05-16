# Bricks!

[![build](https://github.com/databricks/bricks/workflows/build/badge.svg?branch=main)](https://github.com/databricks/bricks/actions?query=workflow%3Abuild+branch%3Amain)

This is an early PoC at this stage!

`make build` (or [download the latest from releases page](https://github.com/databricks/bricks/releases)).

Reuses authentication from Databricks CLI. And terraform provider. See details here: https://registry.terraform.io/providers/databrickslabs/databricks/latest/docs#environment-variables

Supports:
* Databricks CLI
* Databricks CLI Profiles
* Azure CLI Auth
* Azure MSI Auth
* Azure SPN Auth
* Google OIDC Auth
* Direct `DATABRICKS_HOST`, `DATABRICKS_TOKEN` or `DATABRICKS_USERNAME` + `DATABRICKS_PASSWORD` variables.

What works:

* `./bricks fs ls /`
* `./bricks test`
* `./bricks launch test.py`

What doesn't work:

* Everything else.

This project reuses some code from Databricks Terraform Provider