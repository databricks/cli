// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppAppActiveDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppAppActiveDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppActiveDeployment struct {
	CreateTime          string                                               `json:"create_time,omitempty"`
	Creator             string                                               `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                               `json:"deployment_id,omitempty"`
	Mode                string                                               `json:"mode,omitempty"`
	SourceCodePath      string                                               `json:"source_code_path,omitempty"`
	Status              *DataSourceAppAppActiveDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                               `json:"update_time,omitempty"`
}

type DataSourceAppAppAppStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppComputeStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppPendingDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppAppPendingDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppPendingDeployment struct {
	CreateTime          string                                                `json:"create_time,omitempty"`
	Creator             string                                                `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                `json:"deployment_id,omitempty"`
	Mode                string                                                `json:"mode,omitempty"`
	SourceCodePath      string                                                `json:"source_code_path,omitempty"`
	Status              *DataSourceAppAppPendingDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                                `json:"update_time,omitempty"`
}

type DataSourceAppAppResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppAppResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppAppResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppAppResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppAppResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppAppResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppAppResources struct {
	Database        *DataSourceAppAppResourcesDatabase        `json:"database,omitempty"`
	Description     string                                    `json:"description,omitempty"`
	Job             *DataSourceAppAppResourcesJob             `json:"job,omitempty"`
	Name            string                                    `json:"name"`
	Secret          *DataSourceAppAppResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppAppResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppAppResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppAppResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppApp struct {
	ActiveDeployment         *DataSourceAppAppActiveDeployment  `json:"active_deployment,omitempty"`
	AppStatus                *DataSourceAppAppAppStatus         `json:"app_status,omitempty"`
	BudgetPolicyId           string                             `json:"budget_policy_id,omitempty"`
	ComputeStatus            *DataSourceAppAppComputeStatus     `json:"compute_status,omitempty"`
	CreateTime               string                             `json:"create_time,omitempty"`
	Creator                  string                             `json:"creator,omitempty"`
	DefaultSourceCodePath    string                             `json:"default_source_code_path,omitempty"`
	Description              string                             `json:"description,omitempty"`
	EffectiveBudgetPolicyId  string                             `json:"effective_budget_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                           `json:"effective_user_api_scopes,omitempty"`
	Id                       string                             `json:"id,omitempty"`
	Name                     string                             `json:"name"`
	Oauth2AppClientId        string                             `json:"oauth2_app_client_id,omitempty"`
	Oauth2AppIntegrationId   string                             `json:"oauth2_app_integration_id,omitempty"`
	PendingDeployment        *DataSourceAppAppPendingDeployment `json:"pending_deployment,omitempty"`
	Resources                []DataSourceAppAppResources        `json:"resources,omitempty"`
	ServicePrincipalClientId string                             `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                             `json:"service_principal_name,omitempty"`
	UpdateTime               string                             `json:"update_time,omitempty"`
	Updater                  string                             `json:"updater,omitempty"`
	Url                      string                             `json:"url,omitempty"`
	UserApiScopes            []string                           `json:"user_api_scopes,omitempty"`
}

type DataSourceApp struct {
	App  *DataSourceAppApp `json:"app,omitempty"`
	Name string            `json:"name"`
}
