// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSandboxesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSandboxesSandboxesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSandboxesSandboxesSpecCompute struct {
	InactivityTimeout string `json:"inactivity_timeout,omitempty"`
}

type DataSourceSandboxesSandboxesSpec struct {
	Compute *DataSourceSandboxesSandboxesSpecCompute `json:"compute,omitempty"`
}

type DataSourceSandboxesSandboxesStatus struct {
	State string `json:"state,omitempty"`
}

type DataSourceSandboxesSandboxes struct {
	CreateTime     string                                      `json:"create_time,omitempty"`
	DisplayName    string                                      `json:"display_name,omitempty"`
	Name           string                                      `json:"name"`
	ProviderConfig *DataSourceSandboxesSandboxesProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourceSandboxesSandboxesSpec           `json:"spec,omitempty"`
	Status         *DataSourceSandboxesSandboxesStatus         `json:"status,omitempty"`
	UpdateTime     string                                      `json:"update_time,omitempty"`
}

type DataSourceSandboxes struct {
	PageSize       int                                `json:"page_size,omitempty"`
	ProviderConfig *DataSourceSandboxesProviderConfig `json:"provider_config,omitempty"`
	Sandboxes      []DataSourceSandboxesSandboxes     `json:"sandboxes,omitempty"`
}
