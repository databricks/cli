// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAwsCrossaccountPolicy struct {
	AwsAccountId    string   `json:"aws_account_id,omitempty"`
	AwsPartition    string   `json:"aws_partition,omitempty"`
	Id              string   `json:"id,omitempty"`
	Json            string   `json:"json,omitempty"`
	PassRoles       []string `json:"pass_roles,omitempty"`
	PolicyType      string   `json:"policy_type,omitempty"`
	Region          string   `json:"region,omitempty"`
	SecurityGroupId string   `json:"security_group_id,omitempty"`
	VpcId           string   `json:"vpc_id,omitempty"`
}
