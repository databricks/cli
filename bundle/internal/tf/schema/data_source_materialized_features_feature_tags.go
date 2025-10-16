// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMaterializedFeaturesFeatureTagsFeatureTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceMaterializedFeaturesFeatureTags struct {
	FeatureTags []DataSourceMaterializedFeaturesFeatureTagsFeatureTags `json:"feature_tags,omitempty"`
}
