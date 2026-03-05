#!/bin/bash
set -euo pipefail

if [ "$ARCH" != "amd64" ] && [ "$ARCH" != "arm64" ]; then
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Download and install the Databricks CLI
mkdir -p /app
wget https://github.com/databricks/cli/releases/download/v${CLI_VERSION}/databricks_cli_${CLI_VERSION}_linux_${ARCH}.tar.gz -O cli.tar.gz
tar -xzf cli.tar.gz -C /app
rm cli.tar.gz

DATABRICKS_TF_VERSION="1.14.0"
DATABRICKS_TF_PROVIDER_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.providerVersion)

# Download the terraform binary
mkdir -p zip
wget https://releases.hashicorp.com/terraform/${DATABRICKS_TF_VERSION}/terraform_${DATABRICKS_TF_VERSION}_linux_${ARCH}.zip -O zip/terraform.zip

# Unzip the terraform binary.
unzip zip/terraform.zip -d zip/terraform
mkdir -p /app/bin
mv zip/terraform/terraform /app/bin/terraform

# Download the provider plugin
TF_PROVIDER_NAME=terraform-provider-databricks_${DATABRICKS_TF_PROVIDER_VERSION}_linux_${ARCH}.zip
mkdir -p /app/providers/registry.terraform.io/databricks/databricks
wget https://github.com/databricks/terraform-provider-databricks/releases/download/v${DATABRICKS_TF_PROVIDER_VERSION}/${TF_PROVIDER_NAME} -O /app/providers/registry.terraform.io/databricks/databricks/${TF_PROVIDER_NAME}
