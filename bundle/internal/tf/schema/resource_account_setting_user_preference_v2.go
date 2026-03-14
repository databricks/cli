// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountSettingUserPreferenceV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingUserPreferenceV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingUserPreferenceV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingUserPreferenceV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingUserPreferenceV2 struct {
	BooleanVal          *ResourceAccountSettingUserPreferenceV2BooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *ResourceAccountSettingUserPreferenceV2EffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveStringVal  *ResourceAccountSettingUserPreferenceV2EffectiveStringVal  `json:"effective_string_val,omitempty"`
	Name                string                                                     `json:"name,omitempty"`
	StringVal           *ResourceAccountSettingUserPreferenceV2StringVal           `json:"string_val,omitempty"`
	UserId              string                                                     `json:"user_id,omitempty"`
}
