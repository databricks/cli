// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema


type ResourcePostgresDatabaseProviderConfig struct {
    WorkspaceId string `json:"workspace_id"`
}

type ResourcePostgresDatabaseSpec struct {
    PostgresDatabase string `json:"postgres_database,omitempty"`
    Role string `json:"role,omitempty"`
}

type ResourcePostgresDatabaseStatus struct {
    PostgresDatabase string `json:"postgres_database,omitempty"`
    Role string `json:"role,omitempty"`
}

type ResourcePostgresDatabase struct {
    CreateTime string `json:"create_time,omitempty"`
    DatabaseId string `json:"database_id,omitempty"`
    Name string `json:"name,omitempty"`
    Parent string `json:"parent"`
    ProviderConfig *ResourcePostgresDatabaseProviderConfig `json:"provider_config,omitempty"`
    Spec *ResourcePostgresDatabaseSpec `json:"spec,omitempty"`
    Status *ResourcePostgresDatabaseStatus `json:"status,omitempty"`
    UpdateTime string `json:"update_time,omitempty"`
}
