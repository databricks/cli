// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAppActiveDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type ResourceAppActiveDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type ResourceAppActiveDeployment struct {
	CreateTime          string                                          `json:"create_time,omitempty"`
	Creator             string                                          `json:"creator,omitempty"`
	DeploymentArtifacts *ResourceAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                          `json:"deployment_id,omitempty"`
	Mode                string                                          `json:"mode,omitempty"`
	SourceCodePath      string                                          `json:"source_code_path,omitempty"`
	Status              *ResourceAppActiveDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                          `json:"update_time,omitempty"`
}

type ResourceAppAppStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type ResourceAppComputeStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type ResourceAppPendingDeploymentDeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path,omitempty"`
}

type ResourceAppPendingDeploymentStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type ResourceAppPendingDeployment struct {
	CreateTime          string                                           `json:"create_time,omitempty"`
	Creator             string                                           `json:"creator,omitempty"`
	DeploymentArtifacts *ResourceAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	DeploymentId        string                                           `json:"deployment_id,omitempty"`
	Mode                string                                           `json:"mode,omitempty"`
	SourceCodePath      string                                           `json:"source_code_path,omitempty"`
	Status              *ResourceAppPendingDeploymentStatus              `json:"status,omitempty"`
	UpdateTime          string                                           `json:"update_time,omitempty"`
}

type ResourceAppResourcesJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppResourcesSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type ResourceAppResourcesServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type ResourceAppResourcesSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppResources struct {
	Description     string                               `json:"description,omitempty"`
	Job             *ResourceAppResourcesJob             `json:"job,omitempty"`
	Name            string                               `json:"name"`
	Secret          *ResourceAppResourcesSecret          `json:"secret,omitempty"`
	ServingEndpoint *ResourceAppResourcesServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *ResourceAppResourcesSqlWarehouse    `json:"sql_warehouse,omitempty"`
}

type ResourceApp struct {
	ActiveDeployment         *ResourceAppActiveDeployment  `json:"active_deployment,omitempty"`
	AppStatus                *ResourceAppAppStatus         `json:"app_status,omitempty"`
	ComputeStatus            *ResourceAppComputeStatus     `json:"compute_status,omitempty"`
	CreateTime               string                        `json:"create_time,omitempty"`
	Creator                  string                        `json:"creator,omitempty"`
	DefaultSourceCodePath    string                        `json:"default_source_code_path,omitempty"`
	Description              string                        `json:"description,omitempty"`
	Name                     string                        `json:"name"`
	NoCompute                bool                          `json:"no_compute,omitempty"`
	PendingDeployment        *ResourceAppPendingDeployment `json:"pending_deployment,omitempty"`
	Resources                []ResourceAppResources        `json:"resources,omitempty"`
	ServicePrincipalClientId string                        `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                           `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                        `json:"service_principal_name,omitempty"`
	UpdateTime               string                        `json:"update_time,omitempty"`
	Updater                  string                        `json:"updater,omitempty"`
	Url                      string                        `json:"url,omitempty"`
}
