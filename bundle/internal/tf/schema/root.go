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
const ProviderVersion = "1.117.0"
const ProviderChecksumLinuxAmd64 = "d6ed0d2080123d52f2a3161fcf05cf045f1e206c2e0127cad17abd99b65fed44"
const ProviderChecksumLinuxArm64 = "130ef32b6fb74f3da0a4c3a8daf23f1fa58334780e0d066670b966cb86b01187"

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
