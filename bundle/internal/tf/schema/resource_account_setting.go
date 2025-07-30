// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountSettingBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingEffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingEffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingEffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSetting struct {
	BooleanVal          *ResourceAccountSettingBooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *ResourceAccountSettingEffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveIntegerVal *ResourceAccountSettingEffectiveIntegerVal `json:"effective_integer_val,omitempty"`
	EffectiveStringVal  *ResourceAccountSettingEffectiveStringVal  `json:"effective_string_val,omitempty"`
	IntegerVal          *ResourceAccountSettingIntegerVal          `json:"integer_val,omitempty"`
	Name                string                                     `json:"name,omitempty"`
	StringVal           *ResourceAccountSettingStringVal           `json:"string_val,omitempty"`
}
