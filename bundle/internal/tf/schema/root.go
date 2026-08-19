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
	ProviderVersion            = "1.127.0"
	ProviderChecksumLinuxAmd64 = "b60c02a8415398f299eed9df1797f00a5e62c571ebbce03897ac64db8c06820f"
	ProviderChecksumLinuxArm64 = "069f302757ba376e14e8e2c492a3cc146bc6552702baab48c639a766bcf97dd2"
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
