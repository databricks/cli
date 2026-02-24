// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsAppActiveDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppsAppActiveDeploymentEnvVars struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"value_from,omitempty"`
}

type DataSourceAppsAppActiveDeploymentGitSourceGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppsAppActiveDeploymentGitSource struct {
	Branch         string                                                   `json:"branch,omitempty"`
	Commit         string                                                   `json:"commit,omitempty"`
	GitRepository  *DataSourceAppsAppActiveDeploymentGitSourceGitRepository `json:"git_repository,omitempty"`
	ResolvedCommit string                                                   `json:"resolved_commit,omitempty"`
	SourceCodePath string                                                   `json:"source_code_path,omitempty"`
	Tag            string                                                   `json:"tag,omitempty"`
}

type DataSourceAppsAppActiveDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppActiveDeployment struct {
	Command             []string                                              `json:"command,omitempty"`
	CreateTime          string                                                `json:"create_time,omitempty"`
	Creator             string                                                `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppsAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                `json:"deployment_id,omitempty"`
	EnvVars             []DataSourceAppsAppActiveDeploymentEnvVars            `json:"env_vars,omitempty"`
	GitSource           *DataSourceAppsAppActiveDeploymentGitSource           `json:"git_source,omitempty"`
	Mode                string                                                `json:"mode,omitempty"`
	SourceCodePath      string                                                `json:"source_code_path,omitempty"`
	Status              *DataSourceAppsAppActiveDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                                `json:"update_time,omitempty"`
}

type DataSourceAppsAppAppStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppComputeStatus struct {
	ActiveInstances int    `json:"active_instances,omitempty"`
	Message         string `json:"message,omitempty"`
	State           string `json:"state,omitempty"`
}

type DataSourceAppsAppGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppsAppPendingDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppsAppPendingDeploymentEnvVars struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"value_from,omitempty"`
}

type DataSourceAppsAppPendingDeploymentGitSourceGitRepository struct {
	Provider string `json:"provider"`
	Url      string `json:"url"`
}

type DataSourceAppsAppPendingDeploymentGitSource struct {
	Branch         string                                                    `json:"branch,omitempty"`
	Commit         string                                                    `json:"commit,omitempty"`
	GitRepository  *DataSourceAppsAppPendingDeploymentGitSourceGitRepository `json:"git_repository,omitempty"`
	ResolvedCommit string                                                    `json:"resolved_commit,omitempty"`
	SourceCodePath string                                                    `json:"source_code_path,omitempty"`
	Tag            string                                                    `json:"tag,omitempty"`
}

type DataSourceAppsAppPendingDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppPendingDeployment struct {
	Command             []string                                               `json:"command,omitempty"`
	CreateTime          string                                                 `json:"create_time,omitempty"`
	Creator             string                                                 `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppsAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                 `json:"deployment_id,omitempty"`
	EnvVars             []DataSourceAppsAppPendingDeploymentEnvVars            `json:"env_vars,omitempty"`
	GitSource           *DataSourceAppsAppPendingDeploymentGitSource           `json:"git_source,omitempty"`
	Mode                string                                                 `json:"mode,omitempty"`
	SourceCodePath      string                                                 `json:"source_code_path,omitempty"`
	Status              *DataSourceAppsAppPendingDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                                 `json:"update_time,omitempty"`
}

type DataSourceAppsAppResourcesDatabase struct {
	DatabaseName string `json:"database_name"`
	InstanceName string `json:"instance_name"`
	Permission   string `json:"permission"`
}

type DataSourceAppsAppResourcesExperiment struct {
	ExperimentId string `json:"experiment_id"`
	Permission   string `json:"permission"`
}

type DataSourceAppsAppResourcesGenieSpace struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	SpaceId    string `json:"space_id"`
}

type DataSourceAppsAppResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsAppResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type DataSourceAppsAppResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type DataSourceAppsAppResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type DataSourceAppsAppResourcesUcSecurable struct {
	Permission        string `json:"permission"`
	SecurableFullName string `json:"securable_full_name"`
	SecurableType     string `json:"securable_type"`
}

type DataSourceAppsAppResources struct {
	Database        *DataSourceAppsAppResourcesDatabase        `json:"database,omitempty"`
	Description     string                                     `json:"description,omitempty"`
	Experiment      *DataSourceAppsAppResourcesExperiment      `json:"experiment,omitempty"`
	GenieSpace      *DataSourceAppsAppResourcesGenieSpace      `json:"genie_space,omitempty"`
	Job             *DataSourceAppsAppResourcesJob             `json:"job,omitempty"`
	Name            string                                     `json:"name"`
	Secret          *DataSourceAppsAppResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppsAppResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppsAppResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
	UcSecurable     *DataSourceAppsAppResourcesUcSecurable     `json:"uc_securable,omitempty"`
}

type DataSourceAppsApp struct {
	ActiveDeployment         *DataSourceAppsAppActiveDeployment  `json:"active_deployment,omitempty"`
	AppStatus                *DataSourceAppsAppAppStatus         `json:"app_status,omitempty"`
	BudgetPolicyId           string                              `json:"budget_policy_id,omitempty"`
	ComputeSize              string                              `json:"compute_size,omitempty"`
	ComputeStatus            *DataSourceAppsAppComputeStatus     `json:"compute_status,omitempty"`
	CreateTime               string                              `json:"create_time,omitempty"`
	Creator                  string                              `json:"creator,omitempty"`
	DefaultSourceCodePath    string                              `json:"default_source_code_path,omitempty"`
	Description              string                              `json:"description,omitempty"`
	EffectiveBudgetPolicyId  string                              `json:"effective_budget_policy_id,omitempty"`
	EffectiveUsagePolicyId   string                              `json:"effective_usage_policy_id,omitempty"`
	EffectiveUserApiScopes   []string                            `json:"effective_user_api_scopes,omitempty"`
	GitRepository            *DataSourceAppsAppGitRepository     `json:"git_repository,omitempty"`
	Id                       string                              `json:"id,omitempty"`
	Name                     string                              `json:"name"`
	Oauth2AppClientId        string                              `json:"oauth2_app_client_id,omitempty"`
	Oauth2AppIntegrationId   string                              `json:"oauth2_app_integration_id,omitempty"`
	PendingDeployment        *DataSourceAppsAppPendingDeployment `json:"pending_deployment,omitempty"`
	Resources                []DataSourceAppsAppResources        `json:"resources,omitempty"`
	ServicePrincipalClientId string                              `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                 `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                              `json:"service_principal_name,omitempty"`
	Space                    string                              `json:"space,omitempty"`
	UpdateTime               string                              `json:"update_time,omitempty"`
	Updater                  string                              `json:"updater,omitempty"`
	Url                      string                              `json:"url,omitempty"`
	UsagePolicyId            string                              `json:"usage_policy_id,omitempty"`
	UserApiScopes            []string                            `json:"user_api_scopes,omitempty"`
}

type DataSourceAppsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceApps struct {
	App            []DataSourceAppsApp           `json:"app,omitempty"`
	ProviderConfig *DataSourceAppsProviderConfig `json:"provider_config,omitempty"`
}
