// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePermissionsAccessControl struct {
	GroupName            string `json:"group_name,omitempty"`
	PermissionLevel      string `json:"permission_level,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type ResourcePermissions struct {
	AlertV2Id              string                             `json:"alert_v2_id,omitempty"`
	AppName                string                             `json:"app_name,omitempty"`
	Authorization          string                             `json:"authorization,omitempty"`
	ClusterId              string                             `json:"cluster_id,omitempty"`
	ClusterPolicyId        string                             `json:"cluster_policy_id,omitempty"`
	DashboardId            string                             `json:"dashboard_id,omitempty"`
	DatabaseInstanceName   string                             `json:"database_instance_name,omitempty"`
	DirectoryId            string                             `json:"directory_id,omitempty"`
	DirectoryPath          string                             `json:"directory_path,omitempty"`
	ExperimentId           string                             `json:"experiment_id,omitempty"`
	Id                     string                             `json:"id,omitempty"`
	InstancePoolId         string                             `json:"instance_pool_id,omitempty"`
	JobId                  string                             `json:"job_id,omitempty"`
	NotebookId             string                             `json:"notebook_id,omitempty"`
	NotebookPath           string                             `json:"notebook_path,omitempty"`
	ObjectType             string                             `json:"object_type,omitempty"`
	PipelineId             string                             `json:"pipeline_id,omitempty"`
	RegisteredModelId      string                             `json:"registered_model_id,omitempty"`
	RepoId                 string                             `json:"repo_id,omitempty"`
	RepoPath               string                             `json:"repo_path,omitempty"`
	ServingEndpointId      string                             `json:"serving_endpoint_id,omitempty"`
	SqlAlertId             string                             `json:"sql_alert_id,omitempty"`
	SqlDashboardId         string                             `json:"sql_dashboard_id,omitempty"`
	SqlEndpointId          string                             `json:"sql_endpoint_id,omitempty"`
	SqlQueryId             string                             `json:"sql_query_id,omitempty"`
	VectorSearchEndpointId string                             `json:"vector_search_endpoint_id,omitempty"`
	WorkspaceFileId        string                             `json:"workspace_file_id,omitempty"`
	WorkspaceFilePath      string                             `json:"workspace_file_path,omitempty"`
	AccessControl          []ResourcePermissionsAccessControl `json:"access_control,omitempty"`
}
