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
	ProviderVersion            = "1.131.0"
	ProviderChecksumLinuxAmd64 = "f43ef6779b8e13effb98bd553c3d05136d4493c3ca254e9a1ace5c4d845f88f2"
	ProviderChecksumLinuxArm64 = "f8fe28d63dbfedfd8aec0878505b3d4e2e99a294f6c3e3a1c3063c07beb07ce3"
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
