package tf

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

func NewRoot() *Root {
	return &Root{
		Terraform: map[string]interface{}{
			"required_providers": map[string]interface{}{
				"databricks": map[string]interface{}{
					"source":  "databricks/databricks",
					"version": ">= 1.0.0",
				},
			},
		},

		Provider: NewProviders(),
		Data:     NewDataSources(),
		Resource: NewResources(),
	}
}
