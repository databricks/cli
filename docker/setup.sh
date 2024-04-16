#!/bin/bash
set -euo pipefail

DATABRICKS_TF_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.version)
DATABRICKS_TF_PROVIDER_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.providerVersion)

# Checksums for the terraform binary version 1.5.5. The checksums are used to verify the integrity of the downloaded binary.
# The checksums are optained from https://releases.hashicorp.com/terraform/1.5.5/
EXPECTED_CHECKSUM="invalid"
if [ $ARCH == "arm64" ]; then
  EXPECTED_CHECKSUM=b055aefe343d0b710d8a7afd31aeb702b37bbf4493bb9385a709991e48dfbcd2 # linux/arm64
elif [ $ARCH == "amd64" ]; then
  EXPECTED_CHECKSUM=ad0c696c870c8525357b5127680cd79c0bdf58179af9acd091d43b1d6482da4a # linux/amd64
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Download the terraform binary
mkdir -p zip
wget https://releases.hashicorp.com/terraform/${DATABRICKS_TF_VERSION}/terraform_${DATABRICKS_TF_VERSION}_linux_${ARCH}.zip -O zip/terraform.zip

# Verify the checksum
COMPUTED_HASH=$(sha256sum zip/terraform.zip | awk '{ print $1 }')
if [ "$COMPUTED_HASH" != "$EXPECTED_CHECKSUM" ]; then
    echo "Checksum mismatch for terraform binary. Version: $DATABRICKS_TF_VERSION, Arch: $ARCH, Expected checksum: $EXPECTED_CHECKSUM, Computed checksum: $COMPUTED_HASH."
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
