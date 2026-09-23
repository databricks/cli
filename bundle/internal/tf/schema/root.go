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
	ProviderVersion            = "1.133.0"
	ProviderChecksumLinuxAmd64 = "70fb8183957f656ffad94db04b90e780335d942e614cdb3af4d08d78a8e43853"
	ProviderChecksumLinuxArm64 = "db4c858f851b42846b0e8cfedcd67a4f262611b6ed8bfadde55575620c6bcf62"
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
