// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceEnvironmentsDefaultWorkspaceBaseEnvironmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceEnvironmentsDefaultWorkspaceBaseEnvironment struct {
	CpuWorkspaceBaseEnvironment string                                                             `json:"cpu_workspace_base_environment,omitempty"`
	GpuWorkspaceBaseEnvironment string                                                             `json:"gpu_workspace_base_environment,omitempty"`
	Name                        string                                                             `json:"name,omitempty"`
	ProviderConfig              *ResourceEnvironmentsDefaultWorkspaceBaseEnvironmentProviderConfig `json:"provider_config,omitempty"`
}
