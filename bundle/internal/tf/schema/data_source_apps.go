// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsAppActiveDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppsAppActiveDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppActiveDeployment struct {
	CreateTime          string                                                `json:"create_time,omitempty"`
	Creator             string                                                `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppsAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                `json:"deployment_id,omitempty"`
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
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppPendingDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type DataSourceAppsAppPendingDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAppsAppPendingDeployment struct {
	CreateTime          string                                                 `json:"create_time,omitempty"`
	Creator             string                                                 `json:"creator,omitempty"`
	DeploymentArtifacts *DataSourceAppsAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                                 `json:"deployment_id,omitempty"`
	Mode                string                                                 `json:"mode,omitempty"`
	SourceCodePath      string                                                 `json:"source_code_path,omitempty"`
	Status              *DataSourceAppsAppPendingDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                                 `json:"update_time,omitempty"`
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

type DataSourceAppsAppResources struct {
	Description     string                                     `json:"description,omitempty"`
	Job             *DataSourceAppsAppResourcesJob             `json:"job,omitempty"`
	Name            string                                     `json:"name"`
	Secret          *DataSourceAppsAppResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *DataSourceAppsAppResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *DataSourceAppsAppResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
}

type DataSourceAppsApp struct {
	ActiveDeployment         *DataSourceAppsAppActiveDeployment  `json:"active_deployment,omitempty"`
	AppStatus                *DataSourceAppsAppAppStatus         `json:"app_status,omitempty"`
	ComputeStatus            *DataSourceAppsAppComputeStatus     `json:"compute_status,omitempty"`
	CreateTime               string                              `json:"create_time,omitempty"`
	Creator                  string                              `json:"creator,omitempty"`
	DefaultSourceCodePath    string                              `json:"default_source_code_path,omitempty"`
	Description              string                              `json:"description,omitempty"`
	Name                     string                              `json:"name"`
	PendingDeployment        *DataSourceAppsAppPendingDeployment `json:"pending_deployment,omitempty"`
	Resources                []DataSourceAppsAppResources        `json:"resources,omitempty"`
	ServicePrincipalClientId string                              `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                                 `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                              `json:"service_principal_name,omitempty"`
	UpdateTime               string                              `json:"update_time,omitempty"`
	Updater                  string                              `json:"updater,omitempty"`
	Url                      string                              `json:"url,omitempty"`
}

type DataSourceApps struct {
	App []DataSourceAppsApp `json:"app,omitempty"`
}
