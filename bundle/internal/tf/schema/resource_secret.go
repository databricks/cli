// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSecretProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceSecret struct {
	ConfigReference      string                        `json:"config_reference,omitempty"`
	Id                   string                        `json:"id,omitempty"`
	Key                  string                        `json:"key"`
	LastUpdatedTimestamp int                           `json:"last_updated_timestamp,omitempty"`
	Scope                string                        `json:"scope"`
	StringValue          string                        `json:"string_value"`
	ProviderConfig       *ResourceSecretProviderConfig `json:"provider_config,omitempty"`
}
