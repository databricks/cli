// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceSettingBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingEffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingEffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingEffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSetting struct {
	BooleanVal          *DataSourceWorkspaceSettingBooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *DataSourceWorkspaceSettingEffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveIntegerVal *DataSourceWorkspaceSettingEffectiveIntegerVal `json:"effective_integer_val,omitempty"`
	EffectiveStringVal  *DataSourceWorkspaceSettingEffectiveStringVal  `json:"effective_string_val,omitempty"`
	IntegerVal          *DataSourceWorkspaceSettingIntegerVal          `json:"integer_val,omitempty"`
	Name                string                                         `json:"name,omitempty"`
	StringVal           *DataSourceWorkspaceSettingStringVal           `json:"string_val,omitempty"`
}
