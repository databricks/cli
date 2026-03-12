// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRestrictWorkspaceAdminsSettingProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceRestrictWorkspaceAdminsSettingRestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type ResourceRestrictWorkspaceAdminsSetting struct {
	Etag                    string                                                         `json:"etag,omitempty"`
	Id                      string                                                         `json:"id,omitempty"`
	SettingName             string                                                         `json:"setting_name,omitempty"`
	ProviderConfig          *ResourceRestrictWorkspaceAdminsSettingProviderConfig          `json:"provider_config,omitempty"`
	RestrictWorkspaceAdmins *ResourceRestrictWorkspaceAdminsSettingRestrictWorkspaceAdmins `json:"restrict_workspace_admins,omitempty"`
}
