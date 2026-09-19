// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSandboxProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSandboxSpecCompute struct {
	InactivityTimeout string `json:"inactivity_timeout,omitempty"`
}

type DataSourceSandboxSpec struct {
	Compute *DataSourceSandboxSpecCompute `json:"compute,omitempty"`
}

type DataSourceSandboxStatus struct {
	State string `json:"state,omitempty"`
}

type DataSourceSandbox struct {
	CreateTime     string                           `json:"create_time,omitempty"`
	DisplayName    string                           `json:"display_name,omitempty"`
	Name           string                           `json:"name"`
	ProviderConfig *DataSourceSandboxProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourceSandboxSpec           `json:"spec,omitempty"`
	Status         *DataSourceSandboxStatus         `json:"status,omitempty"`
	UpdateTime     string                           `json:"update_time,omitempty"`
}
