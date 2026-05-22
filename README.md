# Databricks CLI

[![build](https://github.com/databricks/cli/workflows/build/badge.svg?branch=main)](https://github.com/databricks/cli/actions?query=workflow%3Abuild+branch%3Amain)

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

## Stability Policy

### Feature stability

Commands and flags are stable by default and will not break within a major version.

Some features are unstable and may change in any MINOR release:

- Commands and flags marked **Beta** or **Private Preview** in their `--help` output.
- Commands in the `databricks experimental` group.

### Versioning

The CLI follows [Semantic Versioning](https://semver.org) (`MAJOR.MINOR.PATCH`):

- `MAJOR` is incremented for breaking changes to **stable** features.
- `MINOR` is incremented for new features and for breaking changes to **unstable** features.
- `PATCH` is incremented for backward-compatible bug fixes, security fixes, and dependency updates.

Databricks may ship a breaking change to a stable feature without a major version bump in exceptional circumstances where waiting for the next major version would itself cause greater harm: an active security incident, a legal or compliance requirement, or a regression introduced in the current major version. Any such exceptional change is announced in the release notes.

### Security patches

Security patches ship on the current release, and on specific older versions listed in [`SECURITY.md`](./SECURITY.md). The CLI does not currently offer a broader long-term support commitment.

## Privacy Notice
Databricks CLI use is subject to the [Databricks License](https://github.com/databricks/cli/blob/main/LICENSE) and [Databricks Privacy Notice](https://www.databricks.com/legal/privacynotice), including any Usage Data provisions.
