// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAwsAssumeRolePolicy struct {
	DatabricksAccountId string `json:"databricks_account_id,omitempty"`
	ExternalId          string `json:"external_id"`
	ForLogDelivery      bool   `json:"for_log_delivery,omitempty"`
	Id                  string `json:"id,omitempty"`
	Json                string `json:"json,omitempty"`
}
