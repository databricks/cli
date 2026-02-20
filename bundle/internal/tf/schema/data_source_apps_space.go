// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsSpaceProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppsSpaceResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppsSpaceResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppsSpaceResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type DataSourceAppsSpaceResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpaceResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppsSpaceResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpaceResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpaceResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppsSpaceResources struct {
	Database        *DataSourceAppsSpaceResourcesDatabase        `json:"database,omitempty"`
	Description     string                                       `json:"description,omitempty"`
	Experiment      *DataSourceAppsSpaceResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppsSpaceResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *DataSourceAppsSpaceResourcesJob             `json:"job,omitempty"`
	Name            string                                       `json:"name"`
	Secret          *DataSourceAppsSpaceResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppsSpaceResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppsSpaceResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppsSpaceResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppsSpaceStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsSpace struct {
	CreateTime               string                             `json:"create_time,omitempty"`
	Creator                  string                             `json:"creator,omitempty"`
	Description              string                             `json:"description,omitempty"`
	EffectiveUsagePolicyId   string                             `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                           `json:"effective_user_api_scopes,omitempty"`
	Id                       string                             `json:"id,omitempty"`
	Name                     string                             `json:"name"`
	Oauth2AppClientId        string                             `json:"oauth2_app_client_id,omitempty"`
	Oauth2AppIntegrationId   string                             `json:"oauth2_app_integration_id,omitempty"`
	ProviderConfig           *DataSourceAppsSpaceProviderConfig `json:"provider_config,omitempty"`
	Resources                []DataSourceAppsSpaceResources     `json:"resources,omitempty"`
	ServicePrincipalClientId string                             `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                             `json:"service_principal_name,omitempty"`
	Status                   *DataSourceAppsSpaceStatus         `json:"status,omitempty"`
	UpdateTime               string                             `json:"update_time,omitempty"`
	Updater                  string                             `json:"updater,omitempty"`
	UsagePolicyId            string                             `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                           `json:"user_api_scopes,omitempty"`
}
