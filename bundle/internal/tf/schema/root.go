// Generated from Databricks Terraform provider schema. DO NOT EDIT.
package schema

type Providers struct {
	Databricks *Config `json:"databricks,omitempty"`
}

func NewProviders() *Providers {
	return &Providers{
		Databricks: &Config{},
	}
}

type Root struct {
	Terraform map[string]any `json:"terraform"`

	Provider *Providers   `json:"provider,omitempty"`
	Data     *DataSources `json:"data,omitempty"`
	Resource *Resources   `json:"resource,omitempty"`
}

const ProviderHost = "registry.terraform.io"
const ProviderSource = "databricks/databricks"
const ProviderVersion = "1.114.0"
const ProviderChecksumLinuxAmd64 = "9ce35b938ca9bdefaab8a6fdab94c5c496af85db1d04eec0d2a54412fe3e9bb1"
const ProviderChecksumLinuxArm64 = "25c950b6c151c9ef7c3e48f5864d9efed222e38590df92cfca33d6af5c6716a7"

func NewRoot() *Root {
	return &Root{
		Terraform: map[string]any{
			"required_providers": map[string]any{
				"databricks": map[string]any{
					"source":  ProviderSource,
					"version": ProviderVersion,
				},
			},
		},
	}
}
