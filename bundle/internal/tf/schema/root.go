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
const ProviderVersion = "1.52.0"

func NewRoot() *Root {
	return &Root{
		Terraform: map[string]interface{}{
			"required_providers": map[string]interface{}{
				"databricks": map[string]interface{}{
					"source":  ProviderSource,
					"version": ProviderVersion,
				},
			},
		},
	}
}
