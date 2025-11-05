// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMaterializedFeaturesFeatureTagsFeatureTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceMaterializedFeaturesFeatureTags struct {
	FeatureName string                                                 `json:"feature_name"`
	FeatureTags []DataSourceMaterializedFeaturesFeatureTagsFeatureTags `json:"feature_tags,omitempty"`
	PageSize    int                                                    `json:"page_size,omitempty"`
	TableName   string                                                 `json:"table_name"`
}
