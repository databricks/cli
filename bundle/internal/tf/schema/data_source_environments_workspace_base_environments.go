// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEnvironmentsWorkspaceBaseEnvironmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceEnvironmentsWorkspaceBaseEnvironmentsWorkspaceBaseEnvironmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceEnvironmentsWorkspaceBaseEnvironmentsWorkspaceBaseEnvironments struct {
	BaseEnvironmentType          string                                                                                  `json:"base_environment_type,omitempty"`
	CreateTime                   string                                                                                  `json:"create_time,omitempty"`
	CreatorUserId                string                                                                                  `json:"creator_user_id,omitempty"`
	DisplayName                  string                                                                                  `json:"display_name,omitempty"`
	EffectiveBaseEnvironmentType string                                                                                  `json:"effective_base_environment_type,omitempty"`
	Filepath                     string                                                                                  `json:"filepath,omitempty"`
	IsDefault                    bool                                                                                    `json:"is_default,omitempty"`
	LastUpdatedUserId            string                                                                                  `json:"last_updated_user_id,omitempty"`
	Message                      string                                                                                  `json:"message,omitempty"`
	Name                         string                                                                                  `json:"name"`
	ProviderConfig               *DataSourceEnvironmentsWorkspaceBaseEnvironmentsWorkspaceBaseEnvironmentsProviderConfig `json:"provider_config,omitempty"`
	Status                       string                                                                                  `json:"status,omitempty"`
	UpdateTime                   string                                                                                  `json:"update_time,omitempty"`
}

type DataSourceEnvironmentsWorkspaceBaseEnvironments struct {
	PageSize                  int                                                                        `json:"page_size,omitempty"`
	ProviderConfig            *DataSourceEnvironmentsWorkspaceBaseEnvironmentsProviderConfig             `json:"provider_config,omitempty"`
	WorkspaceBaseEnvironments []DataSourceEnvironmentsWorkspaceBaseEnvironmentsWorkspaceBaseEnvironments `json:"workspace_base_environments,omitempty"`
}
