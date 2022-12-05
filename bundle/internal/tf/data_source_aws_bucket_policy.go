// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type DataSourceAwsBucketPolicy struct {
	Bucket                string `json:"bucket"`
	DatabricksAccountId   string `json:"databricks_account_id,omitempty"`
	DatabricksE2AccountId string `json:"databricks_e2_account_id,omitempty"`
	FullAccessRole        string `json:"full_access_role,omitempty"`
	Id                    string `json:"id,omitempty"`
	Json                  string `json:"json,omitempty"`
}
