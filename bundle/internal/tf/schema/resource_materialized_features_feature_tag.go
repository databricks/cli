// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMaterializedFeaturesFeatureTagProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceMaterializedFeaturesFeatureTag struct {
	Key            string                                                `json:"key"`
	ProviderConfig *ResourceMaterializedFeaturesFeatureTagProviderConfig `json:"provider_config,omitempty"`
	Value          string                                                `json:"value,omitempty"`
}
