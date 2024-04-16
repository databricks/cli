# Databricks CLI

[![build](https://github.com/databricks/cli/workflows/build/badge.svg?branch=main)](https://github.com/databricks/cli/actions?query=workflow%3Abuild+branch%3Amain)

This project is in Public Preview.

Documentation about the full REST API coverage is available in the [docs folder](docs/commands.md).

Documentation is available at https://docs.databricks.com/dev-tools/cli/databricks-cli.html.

## Installation

This CLI is packaged as a dependency-free binary executable and may be located in any directory.
See https://github.com/databricks/cli/releases for releases and
[the docs pages](https://docs.databricks.com/dev-tools/cli/databricks-cli.html) for
installation instructions.

------
You can use the CLI through a Docker image by pulling from `ghcr.io`. You can find more releases
at: https://github.com/databricks/cli/pkgs/container/cli.
```
docker pull ghcr.io/databricks/cli:latest
```

Example of how to run the CLI using the docker image. [TODO: Link docs here once live.]:
```
docker run -e DATABRICKS_HOST=$YOUR_HOST_URL -e DATABRICKS_TOKEN=$YOUR_TOKEN ghcr.io/databricks/cli:latest current-user me
```

## Authentication

This CLI follows the Databricks Unified Authentication principles.

You can find a detailed description at https://github.com/databricks/databricks-sdk-go#authentication.
