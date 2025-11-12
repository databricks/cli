# Databricks CLI

[![build](https://github.com/databricks/cli/workflows/build/badge.svg?branch=main)](https://github.com/databricks/cli/actions?query=workflow%3Abuild+branch%3Amain)

This project is in Public Preview.

Documentation is available at https://docs.databricks.com/dev-tools/cli/databricks-cli.html.

## Installation

This CLI is packaged as a dependency-free binary executable and may be located in any directory.
See https://github.com/databricks/cli/releases for releases and
the [Databricks documentation](https://docs.databricks.com/en/dev-tools/cli/install.html) for detailed information about installing the CLI.

------
### Homebrew

We maintain a [Homebrew tap](https://github.com/databricks/homebrew-tap) for installing the Databricks CLI. You can find instructions for how to install, upgrade and downgrade the CLI using Homebrew [here](https://github.com/databricks/homebrew-tap/blob/main/README.md).

------
### Docker
You can use the CLI via a Docker image by pulling the image from `ghcr.io`. You can find all available versions
at: https://github.com/databricks/cli/pkgs/container/cli.
```
docker pull ghcr.io/databricks/cli:latest
```

Example of how to run the CLI using the Docker image. More documentation is available at https://docs.databricks.com/dev-tools/bundles/airgapped-environment.html.
```
docker run -e DATABRICKS_HOST=$YOUR_HOST_URL -e DATABRICKS_TOKEN=$YOUR_TOKEN ghcr.io/databricks/cli:latest current-user me
```

## Authentication

This CLI follows the Databricks Unified Authentication principles.

You can find a detailed description at https://github.com/databricks/databricks-sdk-go#authentication.

## Apps Commands

### MCP (Model Context Protocol)

The `databricks apps mcp` command starts an MCP server that provides AI agents with tools to interact with Databricks.

#### Features

- **Databricks Integration**: Query catalogs, schemas, tables, and execute SQL
- **Project Scaffolding**: Generate full-stack TypeScript applications from templates
- **Workspace Tools**: File operations, bash execution, grep, and glob in project directories
- **Sandboxed Execution**: Isolated file and command execution

#### Usage

Start the MCP server:

```bash
databricks apps mcp start --warehouse-id <warehouse-id>
```

Check your environment:

```bash
databricks apps mcp check
```

#### Configuration

The MCP server requires:
- **Warehouse ID**: Databricks SQL warehouse for query execution
- **Databricks Authentication**: Via standard CLI auth (profile, environment variables)

Optional flags:
- `--allow-deployment`: Enable deployment operations
- `--docker-image`: Docker image for validation (default: node:20-alpine)
- `--use-dagger`: Use Dagger for containerized validation (default: true)

#### Examples

```bash
# Basic usage
databricks apps mcp start --warehouse-id abc123

# With deployment enabled
databricks apps mcp start --warehouse-id abc123 --allow-deployment

# With custom Docker image
databricks apps mcp start --warehouse-id abc123 --docker-image node:20-alpine
```

## Privacy Notice
Databricks CLI use is subject to the [Databricks License](https://github.com/databricks/cli/blob/main/LICENSE) and [Databricks Privacy Notice](https://www.databricks.com/legal/privacynotice), including any Usage Data provisions.
