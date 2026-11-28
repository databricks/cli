package jsonschema

import (
	"fmt"
	"reflect"
)

type validateTypeFunc func(v any) error

func validateType(v any, fieldType Type) error {
	validateFunc, ok := validateTypeFuncs[fieldType]
	if !ok {
		return nil
	}
	return validateFunc(v)
}

func validateString(v any) error {
	if _, ok := v.(string); !ok {
		return fmt.Errorf("expected type string, but value is %#v", v)
	}
	return nil
}

func validateBoolean(v any) error {
	if _, ok := v.(bool); !ok {
		return fmt.Errorf("expected type boolean, but value is %#v", v)
	}
	return nil
}

// JSON documents decode every number as float64 while prompts and converted
// properties hold int64, so both validators accept either representation, as
// JSON Schema does: an integer is a number, and 3.0 is an integer.
func validateNumber(v any) error {
	if _, ok := numberValue(v); !ok {
		return fmt.Errorf("expected type float, but value is %#v", v)
	}
	return nil
}

func validateInteger(v any) error {
	rv := reflect.ValueOf(v)
	if rv.CanInt() || rv.CanUint() || (rv.CanFloat() && isIntegerValue(rv.Float())) {
		return nil
	}
	return fmt.Errorf("expected type integer, but value is %#v", v)
}

// numberValue returns v as a float64 if v holds a Go integer or float.
func numberValue(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch {
	case rv.CanInt():
		return float64(rv.Int()), true
	case rv.CanUint():
		return float64(rv.Uint()), true
	case rv.CanFloat():
		return rv.Float(), true
	}
	return 0, false
}

// valuesEqual compares two values for const and enum checks. Numbers compare by
// value, so a const of 3 read from the schema (float64) matches 3 from a prompt (int64).
func valuesEqual(a, b any) bool {
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.CanInt() && rb.CanInt() {
		return ra.Int() == rb.Int()
	}
	fa, okA := numberValue(a)
	fb, okB := numberValue(b)
	if okA && okB {
		return fa == fb
	}
	return a == b
}

var validateTypeFuncs map[Type]validateTypeFunc = map[Type]validateTypeFunc{
	StringType:  validateString,
	BooleanType: validateBoolean,
	IntegerType: validateInteger,
	NumberType:  validateNumber,
}
