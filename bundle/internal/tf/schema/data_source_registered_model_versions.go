// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceRegisteredModelVersionsModelVersionsAliases struct {
	AliasName  string `json:"alias_name,omitempty"`
	VersionNum int    `json:"version_num,omitempty"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesConnection struct {
	ConnectionName string `json:"connection_name,omitempty"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesCredential struct {
	CredentialName string `json:"credential_name,omitempty"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesFunction struct {
	FunctionFullName string `json:"function_full_name"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesTable struct {
	TableFullName string `json:"table_full_name"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependencies struct {
	Connection []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesConnection `json:"connection,omitempty"`
	Credential []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesCredential `json:"credential,omitempty"`
	Function   []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesFunction   `json:"function,omitempty"`
	Table      []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependenciesTable      `json:"table,omitempty"`
}

type DataSourceRegisteredModelVersionsModelVersionsModelVersionDependencies struct {
	Dependencies []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependenciesDependencies `json:"dependencies,omitempty"`
}

type DataSourceRegisteredModelVersionsModelVersions struct {
	Aliases                  []DataSourceRegisteredModelVersionsModelVersionsAliases                  `json:"aliases,omitempty"`
	BrowseOnly               bool                                                                     `json:"browse_only,omitempty"`
	CatalogName              string                                                                   `json:"catalog_name,omitempty"`
	Comment                  string                                                                   `json:"comment,omitempty"`
	CreatedAt                int                                                                      `json:"created_at,omitempty"`
	CreatedBy                string                                                                   `json:"created_by,omitempty"`
	Id                       string                                                                   `json:"id,omitempty"`
	MetastoreId              string                                                                   `json:"metastore_id,omitempty"`
	ModelName                string                                                                   `json:"model_name,omitempty"`
	ModelVersionDependencies []DataSourceRegisteredModelVersionsModelVersionsModelVersionDependencies `json:"model_version_dependencies,omitempty"`
	RunId                    string                                                                   `json:"run_id,omitempty"`
	RunWorkspaceId           int                                                                      `json:"run_workspace_id,omitempty"`
	SchemaName               string                                                                   `json:"schema_name,omitempty"`
	Source                   string                                                                   `json:"source,omitempty"`
	Status                   string                                                                   `json:"status,omitempty"`
	StorageLocation          string                                                                   `json:"storage_location,omitempty"`
	UpdatedAt                int                                                                      `json:"updated_at,omitempty"`
	UpdatedBy                string                                                                   `json:"updated_by,omitempty"`
	Version                  int                                                                      `json:"version,omitempty"`
}

type DataSourceRegisteredModelVersions struct {
	FullName      string                                           `json:"full_name"`
	ModelVersions []DataSourceRegisteredModelVersionsModelVersions `json:"model_versions,omitempty"`
}
