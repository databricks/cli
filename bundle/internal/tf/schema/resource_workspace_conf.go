// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceConfProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceWorkspaceConf struct {
	CustomConfig   map[string]string                    `json:"custom_config,omitempty"`
	Id             string                               `json:"id,omitempty"`
	ProviderConfig *ResourceWorkspaceConfProviderConfig `json:"provider_config,omitempty"`
}
