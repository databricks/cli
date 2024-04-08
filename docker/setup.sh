#!/bin/sh
set -euo pipefail

# TODO: Add assertions that this script is build called from /build



TF_VERSION=$(/build/databricks bundle debug terraform --output json | jq .terraform.version -r)
PROVIDER_VERSION=$(/build/databricks bundle debug terraform --output json | jq .terraform.providerVersion -r)
# BUILD_ARCH="${1:-invalid}"

# TODO: Test this
# if [ "$BUILD_ARCH" = "invalid" ]; then
#   exit 1
# fi

# TODO: add check that build arch is either amd64 or arm64
mkdir -p zip

# Download the terraform binary
wget https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_${BUILD_ARCH}.zip -O zip/terraform.zip
unzip zip/terraform.zip -d zip/terraform

# Download the databricks terraform provider
wget https://github.com/databricks/terraform-provider-databricks/releases/download/v${PROVIDER_VERSION}/terraform-provider-databricks_${PROVIDER_VERSION}_linux_${BUILD_ARCH}.zip -O zip/provider.zip
unzip zip/provider.zip -d zip/provider
