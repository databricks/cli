// Package variable models ucm's top-level `variables:` block. Forked from
// bundle/config/variable with the DAB-specific lookup kinds (clusters, SPNs,
// jobs, ...) dropped — ucm only needs UC-applicable lookups and can add more
// as needed.
package variable

import (
	"fmt"
	"reflect"
)

// VariableValue holds any resolved variable value. Aliased to any because
// complex variables can carry arbitrary shapes.
type VariableValue = any

type VariableType string

const (
	VariableTypeComplex VariableType = "complex"
)

// Values returns all valid VariableType values.
func (VariableType) Values() []VariableType {
	return []VariableType{
		VariableTypeComplex,
	}
}

// TargetVariable aliases Variable so the JSON schema for per-target overrides
// can be adjusted separately — a target block may either provide a full
// Variable definition or just a default value.
type TargetVariable Variable

// Variable is an input variable declared under `variables:` in ucm.yml.
type Variable struct {
	// Type of the variable. Currently only "complex" has semantic effect.
	Type VariableType `json:"type,omitempty"`

	// Default value that makes the variable optional.
	Default VariableValue `json:"default,omitempty"`

	// Description used for documentation / `ucm schema` output.
	Description string `json:"description,omitempty"`

	// Value holds the resolved value. Resolution priority (highest to lowest):
	//
	// 1. Command-line flag `--var="foo=bar"`
	// 2. Environment variable `DATABRICKS_UCM_VAR_<NAME>`
	// 3. Per-target default from `targets.<name>.variables.<name>.default`
	// 4. Root default from `variables.<name>.default`
	// 5. Lookup (resolved later by ResolveVariableReferencesInLookup)
	// 6. Error — the variable is required but unset.
	Value VariableValue `json:"value,omitempty" ucm:"readonly"`

	// Lookup resolves a known UC entity name into its ID at runtime.
	Lookup *Lookup `json:"lookup,omitempty"`
}

// HasDefault reports whether the variable declared a default value.
func (v *Variable) HasDefault() bool {
	return v.Default != nil
}

// HasValue reports whether the variable has already been resolved.
func (v *Variable) HasValue() bool {
	return v.Value != nil
}

// Set assigns val to the variable. Returns an error if a value is already
// assigned or if val's shape is incompatible with the declared type.
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

// IsComplexValued reports whether v.Value is a struct/array/slice/map.
func (v *Variable) IsComplexValued() bool {
	rv := reflect.ValueOf(v.Value)
	switch rv.Kind() {
	case reflect.Struct, reflect.Array, reflect.Slice, reflect.Map:
		return true
	default:
		return false
	}
}

// IsComplex reports whether the declared type is "complex".
func (v *Variable) IsComplex() bool {
	return v.Type == VariableTypeComplex
}
