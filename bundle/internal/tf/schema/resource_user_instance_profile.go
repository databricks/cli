// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceUserInstanceProfileProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceUserInstanceProfile struct {
	Api               string                                     `json:"api,omitempty"`
	Id                string                                     `json:"id,omitempty"`
	InstanceProfileId string                                     `json:"instance_profile_id"`
	UserId            string                                     `json:"user_id"`
	ProviderConfig    *ResourceUserInstanceProfileProviderConfig `json:"provider_config,omitempty"`
}
