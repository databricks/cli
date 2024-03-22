package terraform

import (
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/hashicorp/go-version"
)

const TerraformStateFileName = "terraform.tfstate"
const TerraformConfigFileName = "bundle.tf.json"

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
