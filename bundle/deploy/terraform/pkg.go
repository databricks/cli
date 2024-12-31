package terraform

import (
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/hashicorp/go-version"
)

const (
	TerraformStateFileName  = "terraform.tfstate"
	TerraformConfigFileName = "bundle.tf.json"
)

// Users can provide their own terraform binary and databricks terraform provider by setting the following environment variables.
// This allows users to use the CLI in an air-gapped environments. See the `debug terraform` command.
const (
	TerraformExecPathEnv        = "DATABRICKS_TF_EXEC_PATH"
	TerraformVersionEnv         = "DATABRICKS_TF_VERSION"
	TerraformCliConfigPathEnv   = "DATABRICKS_TF_CLI_CONFIG_FILE"
	TerraformProviderVersionEnv = "DATABRICKS_TF_PROVIDER_VERSION"
)

// Terraform CLI version to use and the corresponding checksums for it. The
// checksums are used to verify the integrity of the downloaded binary. Please
// update the checksums when the Terraform version is updated. The checksums
// were obtained from https://releases.hashicorp.com/terraform/1.5.5.
//
// These hashes are not used inside the CLI. They are only co-located here to be
// output in the "databricks bundle debug terraform" output. Downstream applications
// like the CLI docker image use these checksums to verify the integrity of the
// downloaded Terraform archive.
var TerraformVersion = version.Must(version.NewVersion("1.5.5"))

const (
	checksumLinuxArm64 = "b055aefe343d0b710d8a7afd31aeb702b37bbf4493bb9385a709991e48dfbcd2"
	checksumLinuxAmd64 = "ad0c696c870c8525357b5127680cd79c0bdf58179af9acd091d43b1d6482da4a"
)

type Checksum struct {
	LinuxArm64 string `json:"linux_arm64"`
	LinuxAmd64 string `json:"linux_amd64"`
}

type TerraformMetadata struct {
	Version         string   `json:"version"`
	Checksum        Checksum `json:"checksum"`
	ProviderHost    string   `json:"providerHost"`
	ProviderSource  string   `json:"providerSource"`
	ProviderVersion string   `json:"providerVersion"`
}

func NewTerraformMetadata() *TerraformMetadata {
	return &TerraformMetadata{
		Version: TerraformVersion.String(),
		Checksum: Checksum{
			LinuxArm64: checksumLinuxArm64,
			LinuxAmd64: checksumLinuxAmd64,
		},
		ProviderHost:    schema.ProviderHost,
		ProviderSource:  schema.ProviderSource,
		ProviderVersion: schema.ProviderVersion,
	}
}
