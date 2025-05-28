// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDisableLegacyFeaturesSettingDisableLegacyFeatures struct {
	Value bool `json:"value"`
}

type ResourceDisableLegacyFeaturesSetting struct {
	Etag                  string                                                     `json:"etag,omitempty"`
	Id                    string                                                     `json:"id,omitempty"`
	SettingName           string                                                     `json:"setting_name,omitempty"`
	DisableLegacyFeatures *ResourceDisableLegacyFeaturesSettingDisableLegacyFeatures `json:"disable_legacy_features,omitempty"`
}
