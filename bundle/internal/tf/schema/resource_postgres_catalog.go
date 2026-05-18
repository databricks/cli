// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresCatalogProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourcePostgresCatalogSpec struct {
	Branch                  string `json:"branch,omitempty"`
	CreateDatabaseIfMissing bool   `json:"create_database_if_missing,omitempty"`
	PostgresDatabase        string `json:"postgres_database"`
}

type ResourcePostgresCatalogStatus struct {
	Branch           string `json:"branch,omitempty"`
	CatalogId        string `json:"catalog_id,omitempty"`
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Project          string `json:"project,omitempty"`
}

type ResourcePostgresCatalog struct {
	CatalogId      string                                 `json:"catalog_id"`
	CreateTime     string                                 `json:"create_time,omitempty"`
	Name           string                                 `json:"name,omitempty"`
	ProviderConfig *ResourcePostgresCatalogProviderConfig `json:"provider_config,omitempty"`
	Spec           *ResourcePostgresCatalogSpec           `json:"spec,omitempty"`
	Status         *ResourcePostgresCatalogStatus         `json:"status,omitempty"`
	Uid            string                                 `json:"uid,omitempty"`
	UpdateTime     string                                 `json:"update_time,omitempty"`
}
