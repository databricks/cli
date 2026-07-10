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
	ProviderVersion            = "1.121.0"
	ProviderChecksumLinuxAmd64 = "1548a581a52d3d8af6dd33245a05c7e7efd4975a2f5d80dfb625a26958067fcd"
	ProviderChecksumLinuxArm64 = "f52645353a5625433863e9f698285c1c5ccea8d5ab0265bf6e6ee243559c728f"
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
