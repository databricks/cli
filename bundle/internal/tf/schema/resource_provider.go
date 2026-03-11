// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceProviderProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceProvider struct {
	AuthenticationType  string                          `json:"authentication_type"`
	Comment             string                          `json:"comment,omitempty"`
	Id                  string                          `json:"id,omitempty"`
	Name                string                          `json:"name"`
	RecipientProfileStr string                          `json:"recipient_profile_str"`
	ProviderConfig      *ResourceProviderProviderConfig `json:"provider_config,omitempty"`
}
