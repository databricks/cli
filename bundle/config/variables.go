package config

import (
	"fmt"
)

type VariableType string

// TODO: make destroy work without requiring variables to be entered
const (
	VariableTypeString = VariableType("string")
)

// Input variables for the bundle config
type Variable struct {
	// A default value which then makes the variable optional
	Default *string `json:"default,omitempty"`

	// Type for this variable. Support types are:
	//
	// 1. String
	Type VariableType `json:"type"`

	// Documentation for this input variable
	Description string `json:"description,omitempty"`

	// This field stores the resolved value for the variable. The variable are
	// resolved in the following priority order (from highest to lowest)
	//
	// 1. Command line flag. For example: `--var="foo=bar"`
	// 2. Environemnet variable. eg: BUNDLE_VAR_foo=bar
	// 3. default value defined in bundle config
	// 4. Throw error, since if no default value is defined, then the variable
	//    is required
	Value *string `json:"value,omitempty" bundle:"readonly"`
}

// True if the variable has been assigned a default value. Variables without a
// a default value are by defination required
// TODO: test whether this works with empty strings
func (v *Variable) HasDefault() bool {
	return v.Default != nil
}

// True if variable has already been assigned a value
func (v *Variable) HasValue() bool {
	return v.Value != nil
}

func (v *Variable) Set(val string) error {
	if v.HasValue() {
		return fmt.Errorf("variable has already been assigned value: %s", *v.Value)
	}
	switch v.Type {
	case VariableTypeString:
		v.Value = &val

	default:
		return fmt.Errorf("unsupported type %s", v.Type)
	}
	return nil
}
