// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountSettingBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingEffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingEffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceAccountSettingEffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceAccountSettingStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSetting struct {
	BooleanVal          *DataSourceAccountSettingBooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *DataSourceAccountSettingEffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveIntegerVal *DataSourceAccountSettingEffectiveIntegerVal `json:"effective_integer_val,omitempty"`
	EffectiveStringVal  *DataSourceAccountSettingEffectiveStringVal  `json:"effective_string_val,omitempty"`
	IntegerVal          *DataSourceAccountSettingIntegerVal          `json:"integer_val,omitempty"`
	Name                string                                       `json:"name,omitempty"`
	StringVal           *DataSourceAccountSettingStringVal           `json:"string_val,omitempty"`
}
