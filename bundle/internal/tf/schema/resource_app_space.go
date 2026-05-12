// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAppSpaceProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceAppSpaceResourcesApp struct {
	Name       string `json:"name,omitempty"`
	Permission string `json:"permission,omitempty"`
}

type ResourceAppSpaceResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type ResourceAppSpaceResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type ResourceAppSpaceResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type ResourceAppSpaceResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppSpaceResourcesPostgres struct {
	Branch     string `json:"branch,omitempty"`
	Database   string `json:"database,omitempty"`
	Permission string `json:"permission,omitempty"`
}

type ResourceAppSpaceResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type ResourceAppSpaceResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type ResourceAppSpaceResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppSpaceResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableKind     string `json:"securable_kind,omitempty"`
	SecurableType     string `json:"securable_type"`
}

type ResourceAppSpaceResources struct {
	App             *ResourceAppSpaceResourcesApp             `json:"app,omitempty"`
	Database        *ResourceAppSpaceResourcesDatabase        `json:"database,omitempty"`
	Description     string                                    `json:"description,omitempty"`
	Experiment      *ResourceAppSpaceResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *ResourceAppSpaceResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *ResourceAppSpaceResourcesJob             `json:"job,omitempty"`
	Name            string                                    `json:"name"`
	Postgres        *ResourceAppSpaceResourcesPostgres        `json:"postgres,omitempty"`
	Secret          *ResourceAppSpaceResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *ResourceAppSpaceResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *ResourceAppSpaceResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *ResourceAppSpaceResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type ResourceAppSpaceStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type ResourceAppSpace struct {
	CreateTime               string                          `json:"create_time,omitempty"`
	Creator                  string                          `json:"creator,omitempty"`
	Description              string                          `json:"description,omitempty"`
	EffectiveUsagePolicyId   string                          `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                        `json:"effective_user_api_scopes,omitempty"`
	Id                       string                          `json:"id,omitempty"`
	Name                     string                          `json:"name"`
	ProviderConfig           *ResourceAppSpaceProviderConfig `json:"provider_config,omitempty"`
	Resources                []ResourceAppSpaceResources     `json:"resources,omitempty"`
	ServicePrincipalClientId string                          `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                             `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                          `json:"service_principal_name,omitempty"`
	Status                   *ResourceAppSpaceStatus         `json:"status,omitempty"`
	UpdateTime               string                          `json:"update_time,omitempty"`
	Updater                  string                          `json:"updater,omitempty"`
	UsagePolicyId            string                          `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                        `json:"user_api_scopes,omitempty"`
}
