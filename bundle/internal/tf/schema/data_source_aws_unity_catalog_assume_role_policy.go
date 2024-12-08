// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAwsUnityCatalogAssumeRolePolicy struct {
	AwsAccountId       string `json:"aws_account_id"`
	AwsPartition       string `json:"aws_partition,omitempty"`
	ExternalId         string `json:"external_id"`
	Id                 string `json:"id,omitempty"`
	Json               string `json:"json,omitempty"`
	RoleName           string `json:"role_name"`
	UnityCatalogIamArn string `json:"unity_catalog_iam_arn,omitempty"`
}
