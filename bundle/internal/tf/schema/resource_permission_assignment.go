// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePermissionAssignmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourcePermissionAssignment struct {
	DisplayName          string                                      `json:"display_name,omitempty"`
	GroupName            string                                      `json:"group_name,omitempty"`
	Id                   string                                      `json:"id,omitempty"`
	Permissions          []string                                    `json:"permissions"`
	PrincipalId          int                                         `json:"principal_id,omitempty"`
	ServicePrincipalName string                                      `json:"service_principal_name,omitempty"`
	UserName             string                                      `json:"user_name,omitempty"`
	ProviderConfig       *ResourcePermissionAssignmentProviderConfig `json:"provider_config,omitempty"`
}
