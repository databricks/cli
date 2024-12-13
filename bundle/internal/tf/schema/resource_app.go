// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceApp struct {
	ActiveDeployment         any    `json:"active_deployment,omitempty"`
	AppStatus                any    `json:"app_status,omitempty"`
	ComputeStatus            any    `json:"compute_status,omitempty"`
	CreateTime               string `json:"create_time,omitempty"`
	Creator                  string `json:"creator,omitempty"`
	DefaultSourceCodePath    string `json:"default_source_code_path,omitempty"`
	Description              string `json:"description,omitempty"`
	Name                     string `json:"name"`
	PendingDeployment        any    `json:"pending_deployment,omitempty"`
	Resources                any    `json:"resources,omitempty"`
	ServicePrincipalClientId string `json:"service_principal_client_id,omitempty"`
	ServicePrincipalId       int    `json:"service_principal_id,omitempty"`
	ServicePrincipalName     string `json:"service_principal_name,omitempty"`
	UpdateTime               string `json:"update_time,omitempty"`
	Updater                  string `json:"updater,omitempty"`
	Url                      string `json:"url,omitempty"`
}
