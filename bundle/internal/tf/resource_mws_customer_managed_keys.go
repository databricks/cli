// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceMwsCustomerManagedKeysAwsKeyInfo struct {
	KeyAlias  string `json:"key_alias"`
	KeyArn    string `json:"key_arn"`
	KeyRegion string `json:"key_region,omitempty"`
}

type ResourceMwsCustomerManagedKeys struct {
	AccountId            string                                    `json:"account_id"`
	CreationTime         int                                       `json:"creation_time,omitempty"`
	CustomerManagedKeyId string                                    `json:"customer_managed_key_id,omitempty"`
	Id                   string                                    `json:"id,omitempty"`
	UseCases             []string                                  `json:"use_cases"`
	AwsKeyInfo           *ResourceMwsCustomerManagedKeysAwsKeyInfo `json:"aws_key_info,omitempty"`
}
