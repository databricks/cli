package variable

import (
	"fmt"
	"reflect"
)

// We are using `any` because since introduction of complex variables,
// variables can be of any type.
// Type alias is used to make it easier to understand the code.
type VariableValue = any

type VariableType string

const (
	VariableTypeComplex VariableType = "complex"
)

// We alias it here to override the JSON schema associated with a variable value
// in a target override. This is because we allow for directly specifying the value
// in addition to the variable.Variable struct format in a target override.
type TargetVariable Variable

// An input variable for the bundle config
type Variable struct {
	// A type of the variable. This is used to validate the value of the variable
	Type VariableType `json:"type,omitempty"`

	// A default value which then makes the variable optional
	Default VariableValue `json:"default,omitempty"`

	// Documentation for this input variable
	Description string `json:"description,omitempty"`

	// This field stores the resolved value for the variable. The variable are
	// resolved in the following priority order (from highest to lowest)
	//
	// 1. Command line flag `--var="foo=bar"`
	// 2. Environment variable. eg: BUNDLE_VAR_foo=bar
	// 3. Load defaults from .databricks/bundle/<target>/variable-overrides.json
	// 4. Default value as defined in the applicable targets block
	// 5. Default value defined in variable definition
	// 6. Throw error, since if no default value is defined, then the variable
	//    is required
	Value VariableValue `json:"value,omitempty" bundle:"readonly"`

	// The value of this field will be used to lookup the resource by name
	// And assign the value of the variable to ID of the resource found.
	Lookup *Lookup `json:"lookup,omitempty"`
}

// True if the variable has been assigned a default value. Variables without a
// a default value are by defination required
func (v *Variable) HasDefault() bool {
	return v.Default != nil
}

// True if variable has already been assigned a value
func (v *Variable) HasValue() bool {
	return v.Value != nil
}

func (v *Variable) Set(val VariableValue) error {
	if v.HasValue() {
		return fmt.Errorf("variable has already been assigned value: %s", v.Value)
	}

	if v.IsComplexValued() && v.Type != VariableTypeComplex {
		return fmt.Errorf("variable type is not complex: %s", v.Type)
	}

	v.Value = val

	return nil
}

func (v *Variable) IsComplexValued() bool {
	rv := reflect.ValueOf(v.Value)
	switch rv.Kind() {
	case reflect.Struct, reflect.Array, reflect.Slice, reflect.Map:
		return true
	default:
		return false
	}
}

func (v *Variable) IsComplex() bool {
	return v.Type == VariableTypeComplex
}
