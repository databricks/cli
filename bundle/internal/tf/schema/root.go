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
const ProviderVersion = "1.111.0"
const ProviderChecksumLinuxAmd64 = "c1b46bbaf5c4a0b253309dad072e05025e24731536719d4408bacd48dc0ccfd9"
const ProviderChecksumLinuxArm64 = "ce379c424009b01ec4762dee4d0db27cfc554d921b55a0af8e4203b3652259e9"

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
