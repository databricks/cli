// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountIamWorkspaceAssignmentV2 struct {
	AccountId             string   `json:"account_id,omitempty"`
	EffectiveEntitlements []string `json:"effective_entitlements,omitempty"`
	Entitlements          []string `json:"entitlements,omitempty"`
	PrincipalId           int      `json:"principal_id"`
	PrincipalType         string   `json:"principal_type,omitempty"`
	WorkspaceId           int      `json:"workspace_id,omitempty"`
}
