#!/bin/bash
set -euo pipefail

DATABRICKS_TF_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.version)
DATABRICKS_TF_PROVIDER_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.providerVersion)

if [ $ARCH != "amd64" ] && [ $ARCH != "arm64" ]; then
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Download the terraform binary
mkdir -p zip
wget https://releases.hashicorp.com/terraform/${DATABRICKS_TF_VERSION}/terraform_${DATABRICKS_TF_VERSION}_linux_${ARCH}.zip -O zip/terraform.zip

# Verify the checksum. This is to ensure that the downloaded archive is not tampered with.
EXPECTED_CHECKSUM="$(/app/databricks bundle debug terraform --output json | jq -r .terraform.checksum.linux_$ARCH)"
COMPUTED_CHECKSUM=$(sha256sum zip/terraform.zip | awk '{ print $1 }')
if [ "$COMPUTED_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "Checksum mismatch for Terraform binary. Version: $DATABRICKS_TF_VERSION, Arch: $ARCH, Expected checksum: $EXPECTED_CHECKSUM, Computed checksum: $COMPUTED_CHECKSUM."
    exit 1
fi

# Unzip the terraform binary. It's safe to do so because we have already verified the checksum.
unzip zip/terraform.zip -d zip/terraform
mkdir -p /app/bin
mv zip/terraform/terraform /app/bin/terraform

# Download the provider plugin
TF_PROVIDER_NAME=terraform-provider-databricks_${DATABRICKS_TF_PROVIDER_VERSION}_linux_${ARCH}.zip
mkdir -p /app/providers/registry.terraform.io/databricks/databricks
wget https://github.com/databricks/terraform-provider-databricks/releases/download/v${DATABRICKS_TF_PROVIDER_VERSION}/${TF_PROVIDER_NAME} -O /app/providers/registry.terraform.io/databricks/databricks/${TF_PROVIDER_NAME}
