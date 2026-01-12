// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppAppActiveDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppAppActiveDeploymentEnvVars struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"value_from,omitempty"`
}

type DataSourceAppAppActiveDeploymentGitSourceGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppAppActiveDeploymentGitSource struct {
	Branch         string                                                  `json:"branch,omitempty"`
	Commit         string                                                  `json:"commit,omitempty"`
	GitRepository  *DataSourceAppAppActiveDeploymentGitSourceGitRepository `json:"git_repository,omitempty"`
	ResolvedCommit string                                                  `json:"resolved_commit,omitempty"`
	SourceCodePath string                                                  `json:"source_code_path,omitempty"`
	Tag            string                                                  `json:"tag,omitempty"`
}

type DataSourceAppAppActiveDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppActiveDeployment struct {
	Command             []string                                             `json:"command,omitempty"`
	CreateTime          string                                               `json:"create_time,omitempty"`
	Creator             string                                               `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                               `json:"deployment_id,omitempty"`
	EnvVars             []DataSourceAppAppActiveDeploymentEnvVars            `json:"env_vars,omitempty"`
	GitSource           *DataSourceAppAppActiveDeploymentGitSource           `json:"git_source,omitempty"`
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

type DataSourceAppAppGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppAppPendingDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppAppPendingDeploymentEnvVars struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"value_from,omitempty"`
}

type DataSourceAppAppPendingDeploymentGitSourceGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppAppPendingDeploymentGitSource struct {
	Branch         string                                                   `json:"branch,omitempty"`
	Commit         string                                                   `json:"commit,omitempty"`
	GitRepository  *DataSourceAppAppPendingDeploymentGitSourceGitRepository `json:"git_repository,omitempty"`
	ResolvedCommit string                                                   `json:"resolved_commit,omitempty"`
	SourceCodePath string                                                   `json:"source_code_path,omitempty"`
	Tag            string                                                   `json:"tag,omitempty"`
}

type DataSourceAppAppPendingDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppAppPendingDeployment struct {
	Command             []string                                              `json:"command,omitempty"`
	CreateTime          string                                                `json:"create_time,omitempty"`
	Creator             string                                                `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                `json:"deployment_id,omitempty"`
	EnvVars             []DataSourceAppAppPendingDeploymentEnvVars            `json:"env_vars,omitempty"`
	GitSource           *DataSourceAppAppPendingDeploymentGitSource           `json:"git_source,omitempty"`
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

type DataSourceAppAppResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppAppResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
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
	Experiment      *DataSourceAppAppResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppAppResourcesGenieSpace      `json:"genie_space,omitempty"`
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
	ComputeSize              string                             `json:"compute_size,omitempty"`
	ComputeStatus            *DataSourceAppAppComputeStatus     `json:"compute_status,omitempty"`
	CreateTime               string                             `json:"create_time,omitempty"`
	Creator                  string                             `json:"creator,omitempty"`
	DefaultSourceCodePath    string                             `json:"default_source_code_path,omitempty"`
	Description              string                             `json:"description,omitempty"`
	EffectiveBudgetPolicyId  string                             `json:"effective_budget_policy_id,omitempty"`
	EffectiveUsagePolicyId   string                             `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                           `json:"effective_user_api_scopes,omitempty"`
	GitRepository            *DataSourceAppAppGitRepository     `json:"git_repository,omitempty"`
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
	UsagePolicyId            string                             `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                           `json:"user_api_scopes,omitempty"`
}

type DataSourceAppProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceApp struct {
	App            *DataSourceAppApp            `json:"app,omitempty"`
	Name           string                       `json:"name"`
	ProviderConfig *DataSourceAppProviderConfig `json:"provider_config,omitempty"`
}
