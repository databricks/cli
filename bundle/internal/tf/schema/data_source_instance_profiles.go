// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceInstanceProfilesInstanceProfiles struct {
	Arn     string `json:"arn,omitempty"`
	IsMeta  bool   `json:"is_meta,omitempty"`
	Name    string `json:"name,omitempty"`
	RoleArn string `json:"role_arn,omitempty"`
}

type DataSourceInstanceProfiles struct {
	Id               string                                       `json:"id,omitempty"`
	InstanceProfiles []DataSourceInstanceProfilesInstanceProfiles `json:"instance_profiles,omitempty"`
}
