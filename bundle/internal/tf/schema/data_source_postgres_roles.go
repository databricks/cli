// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresRolesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresRolesRolesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresRolesRolesSpecAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type DataSourcePostgresRolesRolesSpec struct {
	Attributes      *DataSourcePostgresRolesRolesSpecAttributes `json:"attributes,omitempty"`
	AuthMethod      string                                      `json:"auth_method,omitempty"`
	IdentityType    string                                      `json:"identity_type,omitempty"`
	MembershipRoles []string                                    `json:"membership_roles,omitempty"`
	PostgresRole    string                                      `json:"postgres_role,omitempty"`
}

type DataSourcePostgresRolesRolesStatusAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type DataSourcePostgresRolesRolesStatus struct {
	Attributes      *DataSourcePostgresRolesRolesStatusAttributes `json:"attributes,omitempty"`
	AuthMethod      string                                        `json:"auth_method,omitempty"`
	IdentityType    string                                        `json:"identity_type,omitempty"`
	MembershipRoles []string                                      `json:"membership_roles,omitempty"`
	PostgresRole    string                                        `json:"postgres_role,omitempty"`
}

type DataSourcePostgresRolesRoles struct {
	CreateTime     string                                      `json:"create_time,omitempty"`
	Name           string                                      `json:"name"`
	Parent         string                                      `json:"parent,omitempty"`
	ProviderConfig *DataSourcePostgresRolesRolesProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresRolesRolesSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresRolesRolesStatus         `json:"status,omitempty"`
	UpdateTime     string                                      `json:"update_time,omitempty"`
}

type DataSourcePostgresRoles struct {
	PageSize       int                                    `json:"page_size,omitempty"`
	Parent         string                                 `json:"parent"`
	ProviderConfig *DataSourcePostgresRolesProviderConfig `json:"provider_config,omitempty"`
	Roles          []DataSourcePostgresRolesRoles         `json:"roles,omitempty"`
}
