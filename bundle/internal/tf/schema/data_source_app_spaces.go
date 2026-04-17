// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppSpacesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppSpacesSpacesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceAppSpacesSpacesResourcesApp struct {
	Name       string `json:"name,omitempty"`
	Permission string `json:"permission,omitempty"`
}

type DataSourceAppSpacesSpacesResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppSpacesSpacesResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppSpacesSpacesResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type DataSourceAppSpacesSpacesResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppSpacesSpacesResourcesPostgres struct {
	Branch     string `json:"branch,omitempty"`
	Database   string `json:"database,omitempty"`
	Permission string `json:"permission,omitempty"`
}

type DataSourceAppSpacesSpacesResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppSpacesSpacesResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppSpacesSpacesResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppSpacesSpacesResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableKind     string `json:"securable_kind,omitempty"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppSpacesSpacesResources struct {
	App             *DataSourceAppSpacesSpacesResourcesApp             `json:"app,omitempty"`
	Database        *DataSourceAppSpacesSpacesResourcesDatabase        `json:"database,omitempty"`
	Description     string                                             `json:"description,omitempty"`
	Experiment      *DataSourceAppSpacesSpacesResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppSpacesSpacesResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *DataSourceAppSpacesSpacesResourcesJob             `json:"job,omitempty"`
	Name            string                                             `json:"name"`
	Postgres        *DataSourceAppSpacesSpacesResourcesPostgres        `json:"postgres,omitempty"`
	Secret          *DataSourceAppSpacesSpacesResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppSpacesSpacesResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppSpacesSpacesResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppSpacesSpacesResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppSpacesSpacesStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppSpacesSpaces struct {
	CreateTime               string                                   `json:"create_time,omitempty"`
	Creator                  string                                   `json:"creator,omitempty"`
	Description              string                                   `json:"description,omitempty"`
	EffectiveUsagePolicyId   string                                   `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                                 `json:"effective_user_api_scopes,omitempty"`
	Id                       string                                   `json:"id,omitempty"`
	Name                     string                                   `json:"name"`
	ProviderConfig           *DataSourceAppSpacesSpacesProviderConfig `json:"provider_config,omitempty"`
	Resources                []DataSourceAppSpacesSpacesResources     `json:"resources,omitempty"`
	ServicePrincipalClientId string                                   `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                      `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                                   `json:"service_principal_name,omitempty"`
	Status                   *DataSourceAppSpacesSpacesStatus         `json:"status,omitempty"`
	UpdateTime               string                                   `json:"update_time,omitempty"`
	Updater                  string                                   `json:"updater,omitempty"`
	UsagePolicyId            string                                   `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                                 `json:"user_api_scopes,omitempty"`
}

type DataSourceAppSpaces struct {
	PageSize       int                                `json:"page_size,omitempty"`
	ProviderConfig *DataSourceAppSpacesProviderConfig `json:"provider_config,omitempty"`
	Spaces         []DataSourceAppSpacesSpaces        `json:"spaces,omitempty"`
}
