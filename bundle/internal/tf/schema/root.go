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
const ProviderVersion = "1.113.0"
const ProviderChecksumLinuxAmd64 = "4f5caaf7bea4c435ae97c28c45086c213e182b67d1fe9b13f4e91b9e0b6ad7be"
const ProviderChecksumLinuxArm64 = "69693b0bcbab3a184deb2744e8b90d5a9d1f7e19cdc414bc54a87280e37d65a9"

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
