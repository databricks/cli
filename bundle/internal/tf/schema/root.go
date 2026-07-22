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

const (
	ProviderHost               = "registry.terraform.io"
	ProviderSource             = "databricks/databricks"
	ProviderVersion            = "1.122.0"
	ProviderChecksumLinuxAmd64 = "6c01baf771cec6c4a49449146e07040a74dcddb5b9a0cbd5b7697c1f1eba1e6d"
	ProviderChecksumLinuxArm64 = "e336b8c68b4bf44279947cb3324d5fc2a26cdb8d4c3c407602c9933b4d4503e4"
)

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
