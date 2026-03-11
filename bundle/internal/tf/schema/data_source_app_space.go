// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppSpaceProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppSpaceResourcesApp struct {
}

type DataSourceAppSpaceResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppSpaceResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppSpaceResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type DataSourceAppSpaceResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppSpaceResourcesPostgres struct {
	Branch     string `json:"branch,omitempty"`
	Database   string `json:"database,omitempty"`
	Permission string `json:"permission,omitempty"`
}

type DataSourceAppSpaceResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppSpaceResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppSpaceResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppSpaceResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableKind     string `json:"securable_kind,omitempty"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppSpaceResources struct {
	App             *DataSourceAppSpaceResourcesApp             `json:"app,omitempty"`
	Database        *DataSourceAppSpaceResourcesDatabase        `json:"database,omitempty"`
	Description     string                                      `json:"description,omitempty"`
	Experiment      *DataSourceAppSpaceResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppSpaceResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *DataSourceAppSpaceResourcesJob             `json:"job,omitempty"`
	Name            string                                      `json:"name"`
	Postgres        *DataSourceAppSpaceResourcesPostgres        `json:"postgres,omitempty"`
	Secret          *DataSourceAppSpaceResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppSpaceResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppSpaceResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppSpaceResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppSpaceStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppSpace struct {
	CreateTime               string                            `json:"create_time,omitempty"`
	Creator                  string                            `json:"creator,omitempty"`
	Description              string                            `json:"description,omitempty"`
	EffectiveUsagePolicyId   string                            `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                          `json:"effective_user_api_scopes,omitempty"`
	Id                       string                            `json:"id,omitempty"`
	Name                     string                            `json:"name"`
	ProviderConfig           *DataSourceAppSpaceProviderConfig `json:"provider_config,omitempty"`
	Resources                []DataSourceAppSpaceResources     `json:"resources,omitempty"`
	ServicePrincipalClientId string                            `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                               `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                            `json:"service_principal_name,omitempty"`
	Status                   *DataSourceAppSpaceStatus         `json:"status,omitempty"`
	UpdateTime               string                            `json:"update_time,omitempty"`
	Updater                  string                            `json:"updater,omitempty"`
	UsagePolicyId            string                            `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                          `json:"user_api_scopes,omitempty"`
}
