// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceBinding struct {
	BindingType   string `json:"binding_type,omitempty"`
	CatalogName   string `json:"catalog_name,omitempty"`
	Id            string `json:"id,omitempty"`
	SecurableName string `json:"securable_name,omitempty"`
	SecurableType string `json:"securable_type,omitempty"`
	WorkspaceId   int    `json:"workspace_id"`
}
