// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountSettingUserPreferenceV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingUserPreferenceV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingUserPreferenceV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingUserPreferenceV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingUserPreferenceV2 struct {
	BooleanVal          *DataSourceAccountSettingUserPreferenceV2BooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *DataSourceAccountSettingUserPreferenceV2EffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveStringVal  *DataSourceAccountSettingUserPreferenceV2EffectiveStringVal  `json:"effective_string_val,omitempty"`
	Name                string                                                       `json:"name"`
	StringVal           *DataSourceAccountSettingUserPreferenceV2StringVal           `json:"string_val,omitempty"`
	UserId              string                                                       `json:"user_id"`
}
