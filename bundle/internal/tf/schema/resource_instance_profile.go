// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceInstanceProfile struct {
	Id                    string `json:"id,omitempty"`
	InstanceProfileArn    string `json:"instance_profile_arn,omitempty"`
	IsMetaInstanceProfile bool   `json:"is_meta_instance_profile,omitempty"`
	SkipValidation        bool   `json:"skip_validation,omitempty"`
}
