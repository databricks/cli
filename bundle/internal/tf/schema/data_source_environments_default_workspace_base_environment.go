// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEnvironmentsDefaultWorkspaceBaseEnvironmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceEnvironmentsDefaultWorkspaceBaseEnvironment struct {
	CpuWorkspaceBaseEnvironment string                                                               `json:"cpu_workspace_base_environment,omitempty"`
	GpuWorkspaceBaseEnvironment string                                                               `json:"gpu_workspace_base_environment,omitempty"`
	Name                        string                                                               `json:"name"`
	ProviderConfig              *DataSourceEnvironmentsDefaultWorkspaceBaseEnvironmentProviderConfig `json:"provider_config,omitempty"`
}
