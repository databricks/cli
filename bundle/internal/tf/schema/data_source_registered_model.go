// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceRegisteredModelModelInfoAliases struct {
	AliasName  string `json:"alias_name,omitempty"`
	VersionNum int    `json:"version_num,omitempty"`
}

type DataSourceRegisteredModelModelInfo struct {
	Aliases         []DataSourceRegisteredModelModelInfoAliases `json:"aliases,omitempty"`
	BrowseOnly      bool                                        `json:"browse_only,omitempty"`
	CatalogName     string                                      `json:"catalog_name,omitempty"`
	Comment         string                                      `json:"comment,omitempty"`
	CreatedAt       int                                         `json:"created_at,omitempty"`
	CreatedBy       string                                      `json:"created_by,omitempty"`
	FullName        string                                      `json:"full_name,omitempty"`
	MetastoreId     string                                      `json:"metastore_id,omitempty"`
	Name            string                                      `json:"name,omitempty"`
	Owner           string                                      `json:"owner,omitempty"`
	SchemaName      string                                      `json:"schema_name,omitempty"`
	StorageLocation string                                      `json:"storage_location,omitempty"`
	UpdatedAt       int                                         `json:"updated_at,omitempty"`
	UpdatedBy       string                                      `json:"updated_by,omitempty"`
}

type DataSourceRegisteredModel struct {
	FullName       string                               `json:"full_name"`
	IncludeAliases bool                                 `json:"include_aliases,omitempty"`
	IncludeBrowse  bool                                 `json:"include_browse,omitempty"`
	ModelInfo      []DataSourceRegisteredModelModelInfo `json:"model_info,omitempty"`
}
