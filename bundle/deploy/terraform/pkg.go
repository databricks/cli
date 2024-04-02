package terraform

import (
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/hashicorp/go-version"
)

const TerraformStateFileName = "terraform.tfstate"
const TerraformConfigFileName = "bundle.tf.json"

// Users can provide their own terraform binary and databricks terraform provider by setting the following environment variables.
// This allows users to use the CLI in an air-gapped environments. See the `debug terraform` command.
const TerraformExecPathEnv = "DATABRICKS_TF_EXEC_PATH"
const TerraformVersionEnv = "DATABRICKS_TF_VERSION"
const TerraformCliConfigPathEnv = "DATABRICKS_TF_CLI_CONFIG_FILE"
const TerraformProviderVersionEnv = "DATABRICKS_TF_PROVIDER_VERSION"

var TerraformVersion = version.Must(version.NewVersion("1.5.5"))

type TerraformMetadata struct {
	Version         string `json:"version"`
	ProviderHost    string `json:"providerHost"`
	ProviderSource  string `json:"providerSource"`
	ProviderVersion string `json:"providerVersion"`
}

func NewTerraformMetadata() *TerraformMetadata {
	return &TerraformMetadata{
		Version:         TerraformVersion.String(),
		ProviderHost:    schema.ProviderHost,
		ProviderSource:  schema.ProviderSource,
		ProviderVersion: schema.ProviderVersion,
	}
}
