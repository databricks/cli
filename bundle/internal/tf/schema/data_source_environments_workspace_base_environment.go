// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEnvironmentsWorkspaceBaseEnvironmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceEnvironmentsWorkspaceBaseEnvironment struct {
	BaseEnvironmentType          string                                                        `json:"base_environment_type,omitempty"`
	CreateTime                   string                                                        `json:"create_time,omitempty"`
	CreatorUserId                string                                                        `json:"creator_user_id,omitempty"`
	DisplayName                  string                                                        `json:"display_name,omitempty"`
	EffectiveBaseEnvironmentType string                                                        `json:"effective_base_environment_type,omitempty"`
	Filepath                     string                                                        `json:"filepath,omitempty"`
	IsDefault                    bool                                                          `json:"is_default,omitempty"`
	LastUpdatedUserId            string                                                        `json:"last_updated_user_id,omitempty"`
	Message                      string                                                        `json:"message,omitempty"`
	Name                         string                                                        `json:"name"`
	ProviderConfig               *DataSourceEnvironmentsWorkspaceBaseEnvironmentProviderConfig `json:"provider_config,omitempty"`
	Status                       string                                                        `json:"status,omitempty"`
	UpdateTime                   string                                                        `json:"update_time,omitempty"`
}
