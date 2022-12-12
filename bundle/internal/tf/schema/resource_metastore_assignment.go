// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMetastoreAssignment struct {
	DefaultCatalogName string `json:"default_catalog_name,omitempty"`
	Id                 string `json:"id,omitempty"`
	MetastoreId        string `json:"metastore_id"`
	WorkspaceId        int    `json:"workspace_id"`
}
