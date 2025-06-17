package terraform

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/env"
	"github.com/hashicorp/go-version"
)

const (
	TerraformStateFileName  = "terraform.tfstate"
	TerraformConfigFileName = "bundle.tf.json"
)

// Users can provide their own Terraform binary and Databricks Terraform provider by setting the following environment variables.
// This allows users to use the CLI in an air-gapped environments. See the `debug terraform` command.
const (
	TerraformExecPathEnv        = "DATABRICKS_TF_EXEC_PATH"
	TerraformVersionEnv         = "DATABRICKS_TF_VERSION"
	TerraformCliConfigPathEnv   = "DATABRICKS_TF_CLI_CONFIG_FILE"
	TerraformProviderVersionEnv = "DATABRICKS_TF_PROVIDER_VERSION"

	// TerraformVersionOverrideEnv is an environment variable that allows users to override the Terraform version to use.
	// This is useful for testing and development purposes.
	TerraformVersionOverrideEnv = "DATABRICKS_TF_VERSION_OVERRIDE"
)

// TerraformVersion represents the version of the Terraform CLI to use.
// It allows for users overriding the default version.
type TerraformVersion struct {
	Version *version.Version

	// These hashes are not used inside the CLI. They are only co-located here to be
	// output in the "databricks bundle debug terraform" output. Downstream applications
	// like the CLI docker image use these checksums to verify the integrity of the
	// downloaded Terraform archive.
	ChecksumLinuxArm64 string
	ChecksumLinuxAmd64 string
}

// Terraform CLI version to use and the corresponding checksums for it. The
// checksums are used to verify the integrity of the downloaded binary. Please
// update the checksums when the Terraform version is updated. The checksums
// were obtained from https://releases.hashicorp.com/terraform/1.5.5.
var defaultTerraformVersion = TerraformVersion{
	Version: version.Must(version.NewVersion("1.5.5")),

	ChecksumLinuxArm64: "b055aefe343d0b710d8a7afd31aeb702b37bbf4493bb9385a709991e48dfbcd2",
	ChecksumLinuxAmd64: "ad0c696c870c8525357b5127680cd79c0bdf58179af9acd091d43b1d6482da4a",
}

// GetTerraformVersion returns the Terraform version to use.
// The user can configure the Terraform version to use by setting the
// DATABRICKS_TF_VERSION_OVERRIDE environment variable to the desired version.
// It returns an error if the version is malformed.
func GetTerraformVersion(ctx context.Context) (TerraformVersion, error) {
	versionOverride, ok := env.Lookup(ctx, TerraformVersionOverrideEnv)
	if !ok {
		return defaultTerraformVersion, nil
	}

	v, err := version.NewVersion(versionOverride)
	if err != nil {
		return TerraformVersion{}, err
	}

	return TerraformVersion{
		Version: v,
	}, nil
}

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

func NewTerraformMetadata(ctx context.Context) (*TerraformMetadata, error) {
	tv, err := GetTerraformVersion(ctx)
	if err != nil {
		return nil, err
	}
	return &TerraformMetadata{
		Version: tv.Version.String(),
		Checksum: Checksum{
			LinuxArm64: tv.ChecksumLinuxArm64,
			LinuxAmd64: tv.ChecksumLinuxAmd64,
		},
		ProviderHost:    schema.ProviderHost,
		ProviderSource:  schema.ProviderSource,
		ProviderVersion: schema.ProviderVersion,
	}, nil
}
