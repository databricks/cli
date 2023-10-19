package variable

import (
	"fmt"
)

const VariableReferencePrefix = "var"

// An input variable for the bundle config
type Variable struct {
	// A default value which then makes the variable optional
	Default *string `json:"default,omitempty"`

	// Documentation for this input variable
	Description string `json:"description,omitempty"`

	// This field stores the resolved value for the variable. The variable are
	// resolved in the following priority order (from highest to lowest)
	//
	// 1. Command line flag. For example: `--var="foo=bar"`
	// 2. Target variable. eg: BUNDLE_VAR_foo=bar
	// 3. Default value as defined in the applicable environments block
	// 4. Default value defined in variable definition
	// 5. Throw error, since if no default value is defined, then the variable
	//    is required
	Value *string `json:"value,omitempty" bundle:"readonly"`

	// A string value that represents a reference to the remote resource by name
	// Format: "<resource>:<name>"
	// The value of this field will be used to lookup the resource by name
	// And assign the value of the variable to ID of the resource found.
	Lookup string `json:"lookup,omitempty"`
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

func (v *Variable) Set(val string) error {
	if v.HasValue() {
		return fmt.Errorf("variable has already been assigned value: %s", *v.Value)
	}
	v.Value = &val
	return nil
}
