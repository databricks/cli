// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMaterializedFeaturesFeatureTagProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMaterializedFeaturesFeatureTag struct {
	Key            string                                                  `json:"key"`
	ProviderConfig *DataSourceMaterializedFeaturesFeatureTagProviderConfig `json:"provider_config,omitempty"`
	Value          string                                                  `json:"value,omitempty"`
}
