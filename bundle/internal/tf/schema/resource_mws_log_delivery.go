// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsLogDelivery struct {
	AccountId              string `json:"account_id"`
	ConfigId               string `json:"config_id,omitempty"`
	ConfigName             string `json:"config_name,omitempty"`
	CredentialsId          string `json:"credentials_id"`
	DeliveryPathPrefix     string `json:"delivery_path_prefix,omitempty"`
	DeliveryStartTime      string `json:"delivery_start_time,omitempty"`
	Id                     string `json:"id,omitempty"`
	LogType                string `json:"log_type"`
	OutputFormat           string `json:"output_format"`
	Status                 string `json:"status,omitempty"`
	StorageConfigurationId string `json:"storage_configuration_id"`
	WorkspaceIdsFilter     []int  `json:"workspace_ids_filter,omitempty"`
}
