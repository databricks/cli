#!/bin/bash
set -euo pipefail

# Hardcode Terraform version to v1.13.4
DATABRICKS_TF_VERSION="1.13.4"
DATABRICKS_TF_PROVIDER_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.providerVersion)

if [ $ARCH != "amd64" ] && [ $ARCH != "arm64" ]; then
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Download the terraform binary
mkdir -p zip
wget https://releases.hashicorp.com/terraform/${DATABRICKS_TF_VERSION}/terraform_${DATABRICKS_TF_VERSION}_linux_${ARCH}.zip -O zip/terraform.zip

# Verify the checksum. This is to ensure that the downloaded archive is not tampered with.
# Hardcoded checksums for Terraform v1.13.4
if [ $ARCH = "amd64" ]; then
    EXPECTED_CHECKSUM="98aa516201e948306698efd9954ab4cc0d1227c2578ba56245898b5f679e590b"
elif [ $ARCH = "arm64" ]; then
    EXPECTED_CHECKSUM="a17bde150a4d6c9e7ece063ab634c07723b8242e078f3ae9017486277d6690c4"
fi
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
