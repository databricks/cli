// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamWorkspaceAssignmentsV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamWorkspaceAssignmentsV2WorkspaceAssignmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamWorkspaceAssignmentsV2WorkspaceAssignments struct {
	AccountId             string                                                                          `json:"account_id,omitempty"`
	EffectiveEntitlements []string                                                                        `json:"effective_entitlements,omitempty"`
	Entitlements          []string                                                                        `json:"entitlements,omitempty"`
	PrincipalId           int                                                                             `json:"principal_id"`
	PrincipalType         string                                                                          `json:"principal_type,omitempty"`
	ProviderConfig        *DataSourceWorkspaceIamWorkspaceAssignmentsV2WorkspaceAssignmentsProviderConfig `json:"provider_config,omitempty"`
	WorkspaceId           int                                                                             `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamWorkspaceAssignmentsV2 struct {
	PageSize             int                                                                `json:"page_size,omitempty"`
	ProviderConfig       *DataSourceWorkspaceIamWorkspaceAssignmentsV2ProviderConfig        `json:"provider_config,omitempty"`
	WorkspaceAssignments []DataSourceWorkspaceIamWorkspaceAssignmentsV2WorkspaceAssignments `json:"workspace_assignments,omitempty"`
}
