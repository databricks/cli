// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDisableLegacyDbfsSettingDisableLegacyDbfs struct {
	Value bool `json:"value"`
}

type ResourceDisableLegacyDbfsSetting struct {
	Etag              string                                             `json:"etag,omitempty"`
	Id                string                                             `json:"id,omitempty"`
	SettingName       string                                             `json:"setting_name,omitempty"`
	DisableLegacyDbfs *ResourceDisableLegacyDbfsSettingDisableLegacyDbfs `json:"disable_legacy_dbfs,omitempty"`
}
