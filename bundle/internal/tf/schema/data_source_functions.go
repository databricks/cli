// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFunctions struct {
	CatalogName   string `json:"catalog_name"`
	Functions     any    `json:"functions,omitempty"`
	IncludeBrowse bool   `json:"include_browse,omitempty"`
	SchemaName    string `json:"schema_name"`
}
