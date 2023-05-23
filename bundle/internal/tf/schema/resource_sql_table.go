// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlTableColumn struct {
	Comment  string `json:"comment,omitempty"`
	Name     string `json:"name"`
	Nullable bool   `json:"nullable,omitempty"`
	Type     string `json:"type"`
}

type ResourceSqlTable struct {
	CatalogName           string                   `json:"catalog_name"`
	ClusterId             string                   `json:"cluster_id,omitempty"`
	Comment               string                   `json:"comment,omitempty"`
	DataSourceFormat      string                   `json:"data_source_format,omitempty"`
	Id                    string                   `json:"id,omitempty"`
	Name                  string                   `json:"name"`
	Properties            map[string]string        `json:"properties,omitempty"`
	SchemaName            string                   `json:"schema_name"`
	StorageCredentialName string                   `json:"storage_credential_name,omitempty"`
	StorageLocation       string                   `json:"storage_location,omitempty"`
	TableType             string                   `json:"table_type"`
	ViewDefinition        string                   `json:"view_definition,omitempty"`
	Column                []ResourceSqlTableColumn `json:"column,omitempty"`
}
