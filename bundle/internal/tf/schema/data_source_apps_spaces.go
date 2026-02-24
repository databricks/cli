// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsSpacesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppsSpacesSpacesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppsSpacesSpacesResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppsSpacesSpacesResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppsSpacesSpacesResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type DataSourceAppsSpacesSpacesResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpacesSpacesResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppsSpacesSpacesResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpacesSpacesResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsSpacesSpacesResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppsSpacesSpacesResources struct {
	Database        *DataSourceAppsSpacesSpacesResourcesDatabase        `json:"database,omitempty"`
	Description     string                                              `json:"description,omitempty"`
	Experiment      *DataSourceAppsSpacesSpacesResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppsSpacesSpacesResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *DataSourceAppsSpacesSpacesResourcesJob             `json:"job,omitempty"`
	Name            string                                              `json:"name"`
	Secret          *DataSourceAppsSpacesSpacesResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppsSpacesSpacesResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppsSpacesSpacesResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppsSpacesSpacesResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppsSpacesSpacesStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsSpacesSpaces struct {
	CreateTime               string                                    `json:"create_time,omitempty"`
	Creator                  string                                    `json:"creator,omitempty"`
	Description              string                                    `json:"description,omitempty"`
	EffectiveUsagePolicyId   string                                    `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                                  `json:"effective_user_api_scopes,omitempty"`
	Id                       string                                    `json:"id,omitempty"`
	Name                     string                                    `json:"name"`
	Oauth2AppClientId        string                                    `json:"oauth2_app_client_id,omitempty"`
	Oauth2AppIntegrationId   string                                    `json:"oauth2_app_integration_id,omitempty"`
	ProviderConfig           *DataSourceAppsSpacesSpacesProviderConfig `json:"provider_config,omitempty"`
	Resources                []DataSourceAppsSpacesSpacesResources     `json:"resources,omitempty"`
	ServicePrincipalClientId string                                    `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                       `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                                    `json:"service_principal_name,omitempty"`
	Status                   *DataSourceAppsSpacesSpacesStatus         `json:"status,omitempty"`
	UpdateTime               string                                    `json:"update_time,omitempty"`
	Updater                  string                                    `json:"updater,omitempty"`
	UsagePolicyId            string                                    `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                                  `json:"user_api_scopes,omitempty"`
}

type DataSourceAppsSpaces struct {
	PageSize       int                                 `json:"page_size,omitempty"`
	ProviderConfig *DataSourceAppsSpacesProviderConfig `json:"provider_config,omitempty"`
	Spaces         []DataSourceAppsSpacesSpaces        `json:"spaces,omitempty"`
}
