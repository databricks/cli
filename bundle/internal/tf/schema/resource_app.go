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
	DeploymentId        string                                          `json:"deployment_id,omitempty"`
	Mode                string                                          `json:"mode,omitempty"`
	SourceCodePath      string                                          `json:"source_code_path,omitempty"`
	UpdateTime          string                                          `json:"update_time,omitempty"`
	DeploymentArtifacts *ResourceAppActiveDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	Status              *ResourceAppActiveDeploymentStatus              `json:"status,omitempty"`
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
	DeploymentId        string                                           `json:"deployment_id,omitempty"`
	Mode                string                                           `json:"mode,omitempty"`
	SourceCodePath      string                                           `json:"source_code_path,omitempty"`
	UpdateTime          string                                           `json:"update_time,omitempty"`
	DeploymentArtifacts *ResourceAppPendingDeploymentDeploymentArtifacts `json:"deployment_artifacts,omitempty"`
	Status              *ResourceAppPendingDeploymentStatus              `json:"status,omitempty"`
}

type ResourceAppResourceJob struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppResourceSecret struct {
	Key        string `json:"key"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type ResourceAppResourceServingEndpoint struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

type ResourceAppResourceSqlWarehouse struct {
	Id         string `json:"id"`
	Permission string `json:"permission"`
}

type ResourceAppResource struct {
	Description     string                              `json:"description,omitempty"`
	Name            string                              `json:"name"`
	Job             *ResourceAppResourceJob             `json:"job,omitempty"`
	Secret          *ResourceAppResourceSecret          `json:"secret,omitempty"`
	ServingEndpoint *ResourceAppResourceServingEndpoint `json:"serving_endpoint,omitempty"`
	SqlWarehouse    *ResourceAppResourceSqlWarehouse    `json:"sql_warehouse,omitempty"`
}

type ResourceApp struct {
	CreateTime               string                        `json:"create_time,omitempty"`
	Creator                  string                        `json:"creator,omitempty"`
	DefaultSourceCodePath    string                        `json:"default_source_code_path,omitempty"`
	Description              string                        `json:"description,omitempty"`
	Id                       string                        `json:"id,omitempty"`
	Name                     string                        `json:"name"`
	ServicePrincipalClientId string                        `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int                           `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string                        `json:"service_principal_name,omitempty"`
	UpdateTime               string                        `json:"update_time,omitempty"`
	Updater                  string                        `json:"updater,omitempty"`
	Url                      string                        `json:"url,omitempty"`
	ActiveDeployment         *ResourceAppActiveDeployment  `json:"active_deployment,omitempty"`
	AppStatus                *ResourceAppAppStatus         `json:"app_status,omitempty"`
	ComputeStatus            *ResourceAppComputeStatus     `json:"compute_status,omitempty"`
	PendingDeployment        *ResourceAppPendingDeployment `json:"pending_deployment,omitempty"`
	Resource                 []ResourceAppResource         `json:"resource,omitempty"`
}
