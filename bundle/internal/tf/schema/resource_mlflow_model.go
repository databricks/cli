// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMlflowModelProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceMlflowModelTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ResourceMlflowModel struct {
	Description       string                             `json:"description,omitempty"`
	Id                string                             `json:"id,omitempty"`
	Name              string                             `json:"name"`
	RegisteredModelId string                             `json:"registered_model_id,omitempty"`
	ProviderConfig    *ResourceMlflowModelProviderConfig `json:"provider_config,omitempty"`
	Tags              []ResourceMlflowModelTags          `json:"tags,omitempty"`
}
