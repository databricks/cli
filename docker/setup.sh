#!/bin/sh
set -euo pipefail

DATABRICKS_TF_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.version)
DATABRICKS_TF_PROVIDER_VERSION=$(/app/databricks bundle debug terraform --output json | jq -r .terraform.providerVersion)

# Download the terraform binary
mkdir -p zip
wget https://releases.hashicorp.com/terraform/${DATABRICKS_TF_VERSION}/terraform_${DATABRICKS_TF_VERSION}_linux_${ARCH}.zip -O zip/terraform.zip
unzip zip/terraform.zip -d zip/terraform
mkdir -p /app/bin
mv zip/terraform/terraform /app/bin/terraform

# Download the provider plugin
TF_PROVIDER_NAME=terraform-provider-databricks_${DATABRICKS_TF_PROVIDER_VERSION}_linux_${ARCH}.zip
mkdir -p /app/providers/registry.terraform.io/databricks/databricks
wget https://github.com/databricks/terraform-provider-databricks/releases/download/v${DATABRICKS_TF_PROVIDER_VERSION}/${TF_PROVIDER_NAME} -O /app/providers/registry.terraform.io/databricks/databricks/${TF_PROVIDER_NAME}


# TODO: document both the interactive and the non interactive workflows for
# working with DABs with the docker container. Execing into the container
# allows for a better iteration loop

# For the interactive devloop:
# docker run -it --entrypoint /bin/sh cli ...

# For the non-interactive devloop:
# docker run cli ...

# TODO: End to end test for this image?

# TODO: Final sanity check that the docker image is indeed airgapped.
