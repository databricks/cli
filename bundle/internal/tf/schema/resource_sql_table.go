// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlTableColumn struct {
	Comment  string `json:"comment,omitempty"`
	Identity string `json:"identity,omitempty"`
	Name     string `json:"name"`
	Nullable bool   `json:"nullable,omitempty"`
	Type     string `json:"type,omitempty"`
	TypeJson string `json:"type_json,omitempty"`
}

type ResourceSqlTable struct {
	CatalogName           string                   `json:"catalog_name"`
	ClusterId             string                   `json:"cluster_id,omitempty"`
	ClusterKeys           []string                 `json:"cluster_keys,omitempty"`
	Comment               string                   `json:"comment,omitempty"`
	DataSourceFormat      string                   `json:"data_source_format,omitempty"`
	EffectiveProperties   map[string]string        `json:"effective_properties,omitempty"`
	Id                    string                   `json:"id,omitempty"`
	Name                  string                   `json:"name"`
	Options               map[string]string        `json:"options,omitempty"`
	Owner                 string                   `json:"owner,omitempty"`
	Partitions            []string                 `json:"partitions,omitempty"`
	Properties            map[string]string        `json:"properties,omitempty"`
	SchemaName            string                   `json:"schema_name"`
	StorageCredentialName string                   `json:"storage_credential_name,omitempty"`
	StorageLocation       string                   `json:"storage_location,omitempty"`
	TableType             string                   `json:"table_type"`
	ViewDefinition        string                   `json:"view_definition,omitempty"`
	WarehouseId           string                   `json:"warehouse_id,omitempty"`
	Column                []ResourceSqlTableColumn `json:"column,omitempty"`
}
