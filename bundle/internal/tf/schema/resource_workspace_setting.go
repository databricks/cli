// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceSettingBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceWorkspaceSettingEffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceWorkspaceSettingEffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingEffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSetting struct {
	BooleanVal          *ResourceWorkspaceSettingBooleanVal          `json:"boolean_val,omitempty"`
	EffectiveBooleanVal *ResourceWorkspaceSettingEffectiveBooleanVal `json:"effective_boolean_val,omitempty"`
	EffectiveIntegerVal *ResourceWorkspaceSettingEffectiveIntegerVal `json:"effective_integer_val,omitempty"`
	EffectiveStringVal  *ResourceWorkspaceSettingEffectiveStringVal  `json:"effective_string_val,omitempty"`
	IntegerVal          *ResourceWorkspaceSettingIntegerVal          `json:"integer_val,omitempty"`
	Name                string                                       `json:"name,omitempty"`
	StringVal           *ResourceWorkspaceSettingStringVal           `json:"string_val,omitempty"`
}
