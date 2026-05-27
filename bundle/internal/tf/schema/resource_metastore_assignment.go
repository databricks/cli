// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMetastoreAssignmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceMetastoreAssignment struct {
	Api                string                                     `json:"api,omitempty"`
	DefaultCatalogName string                                     `json:"default_catalog_name,omitempty"`
	Id                 string                                     `json:"id,omitempty"`
	MetastoreId        string                                     `json:"metastore_id"`
	WorkspaceId        int                                        `json:"workspace_id"`
	ProviderConfig     *ResourceMetastoreAssignmentProviderConfig `json:"provider_config,omitempty"`
}
