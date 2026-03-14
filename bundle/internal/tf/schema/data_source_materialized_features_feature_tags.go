// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMaterializedFeaturesFeatureTagsFeatureTagsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMaterializedFeaturesFeatureTagsFeatureTags struct {
	Key            string                                                              `json:"key"`
	ProviderConfig *DataSourceMaterializedFeaturesFeatureTagsFeatureTagsProviderConfig `json:"provider_config,omitempty"`
	Value          string                                                              `json:"value,omitempty"`
}

type DataSourceMaterializedFeaturesFeatureTagsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMaterializedFeaturesFeatureTags struct {
	FeatureName    string                                                   `json:"feature_name"`
	FeatureTags    []DataSourceMaterializedFeaturesFeatureTagsFeatureTags   `json:"feature_tags,omitempty"`
	PageSize       int                                                      `json:"page_size,omitempty"`
	ProviderConfig *DataSourceMaterializedFeaturesFeatureTagsProviderConfig `json:"provider_config,omitempty"`
	TableName      string                                                   `json:"table_name"`
}
