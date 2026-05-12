// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGroupInstanceProfileProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceGroupInstanceProfile struct {
	Api               string                                      `json:"api,omitempty"`
	GroupId           string                                      `json:"group_id"`
	Id                string                                      `json:"id,omitempty"`
	InstanceProfileId string                                      `json:"instance_profile_id"`
	ProviderConfig    *ResourceGroupInstanceProfileProviderConfig `json:"provider_config,omitempty"`
}
