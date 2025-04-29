// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDisableLegacyAccessSettingDisableLegacyAccess struct {
	Value bool `json:"value"`
}

type ResourceDisableLegacyAccessSetting struct {
	Etag                string                                                 `json:"etag,omitempty"`
	Id                  string                                                 `json:"id,omitempty"`
	SettingName         string                                                 `json:"setting_name,omitempty"`
	DisableLegacyAccess *ResourceDisableLegacyAccessSettingDisableLegacyAccess `json:"disable_legacy_access,omitempty"`
}
