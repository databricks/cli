// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlGlobalConfig struct {
	DataAccessConfig        map[string]string `json:"data_access_config,omitempty"`
	EnableServerlessCompute bool              `json:"enable_serverless_compute,omitempty"`
	GoogleServiceAccount    string            `json:"google_service_account,omitempty"`
	Id                      string            `json:"id,omitempty"`
	InstanceProfileArn      string            `json:"instance_profile_arn,omitempty"`
	SecurityPolicy          string            `json:"security_policy,omitempty"`
	SqlConfigParams         map[string]string `json:"sql_config_params,omitempty"`
}
