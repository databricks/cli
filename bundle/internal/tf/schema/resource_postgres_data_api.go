// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresDataApiProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourcePostgresDataApiSpec struct {
	DbAggregatesEnabled      bool     `json:"db_aggregates_enabled,omitempty"`
	DbExtraSearchPath        []string `json:"db_extra_search_path,omitempty"`
	DbMaxRows                int      `json:"db_max_rows,omitempty"`
	DbSchemas                []string `json:"db_schemas,omitempty"`
	JwtCacheMaxLifetime      string   `json:"jwt_cache_max_lifetime,omitempty"`
	JwtRoleClaimKey          string   `json:"jwt_role_claim_key,omitempty"`
	OpenapiMode              string   `json:"openapi_mode,omitempty"`
	ServerCorsAllowedOrigins []string `json:"server_cors_allowed_origins,omitempty"`
	ServerTimingEnabled      bool     `json:"server_timing_enabled,omitempty"`
}

type ResourcePostgresDataApiStatus struct {
	AvailableSchemas         []string `json:"available_schemas,omitempty"`
	DbAggregatesEnabled      bool     `json:"db_aggregates_enabled,omitempty"`
	DbExtraSearchPath        []string `json:"db_extra_search_path,omitempty"`
	DbMaxRows                int      `json:"db_max_rows,omitempty"`
	DbSchemas                []string `json:"db_schemas,omitempty"`
	JwtCacheMaxLifetime      string   `json:"jwt_cache_max_lifetime,omitempty"`
	JwtRoleClaimKey          string   `json:"jwt_role_claim_key,omitempty"`
	OpenapiMode              string   `json:"openapi_mode,omitempty"`
	ServerCorsAllowedOrigins []string `json:"server_cors_allowed_origins,omitempty"`
	ServerTimingEnabled      bool     `json:"server_timing_enabled,omitempty"`
	Url                      string   `json:"url,omitempty"`
}

type ResourcePostgresDataApi struct {
	CreateTime     string                                 `json:"create_time,omitempty"`
	Name           string                                 `json:"name,omitempty"`
	Parent         string                                 `json:"parent"`
	ProviderConfig *ResourcePostgresDataApiProviderConfig `json:"provider_config,omitempty"`
	Spec           *ResourcePostgresDataApiSpec           `json:"spec,omitempty"`
	Status         *ResourcePostgresDataApiStatus         `json:"status,omitempty"`
	UpdateTime     string                                 `json:"update_time,omitempty"`
}
