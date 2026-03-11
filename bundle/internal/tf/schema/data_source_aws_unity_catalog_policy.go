// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAwsUnityCatalogPolicy struct {
	AwsAccountId string `json:"aws_account_id"`
	AwsPartition string `json:"aws_partition,omitempty"`
	BucketName   string `json:"bucket_name"`
	Id           string `json:"id,omitempty"`
	Json         string `json:"json,omitempty"`
	KmsName      string `json:"kms_name,omitempty"`
	RoleName     string `json:"role_name"`
}
