// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRegisteredModelAliases struct {
	AliasName   string `json:"alias_name,omitempty"`
	CatalogName string `json:"catalog_name,omitempty"`
	Id          string `json:"id,omitempty"`
	ModelName   string `json:"model_name,omitempty"`
	SchemaName  string `json:"schema_name,omitempty"`
	VersionNum  int    `json:"version_num,omitempty"`
}

type ResourceRegisteredModel struct {
	BrowseOnly      bool                             `json:"browse_only,omitempty"`
	CatalogName     string                           `json:"catalog_name,omitempty"`
	Comment         string                           `json:"comment,omitempty"`
	CreatedAt       int                              `json:"created_at,omitempty"`
	CreatedBy       string                           `json:"created_by,omitempty"`
	FullName        string                           `json:"full_name,omitempty"`
	Id              string                           `json:"id,omitempty"`
	MetastoreId     string                           `json:"metastore_id,omitempty"`
	Name            string                           `json:"name,omitempty"`
	Owner           string                           `json:"owner,omitempty"`
	SchemaName      string                           `json:"schema_name,omitempty"`
	StorageLocation string                           `json:"storage_location,omitempty"`
	UpdatedAt       int                              `json:"updated_at,omitempty"`
	UpdatedBy       string                           `json:"updated_by,omitempty"`
	Aliases         []ResourceRegisteredModelAliases `json:"aliases,omitempty"`
}
