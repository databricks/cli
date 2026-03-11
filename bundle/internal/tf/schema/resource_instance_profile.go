// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceInstanceProfileProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceInstanceProfile struct {
	IamRoleArn            string                                 `json:"iam_role_arn,omitempty"`
	Id                    string                                 `json:"id,omitempty"`
	InstanceProfileArn    string                                 `json:"instance_profile_arn"`
	IsMetaInstanceProfile bool                                   `json:"is_meta_instance_profile,omitempty"`
	SkipValidation        bool                                   `json:"skip_validation,omitempty"`
	ProviderConfig        *ResourceInstanceProfileProviderConfig `json:"provider_config,omitempty"`
}
