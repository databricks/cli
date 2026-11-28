package jsonschema

import (
	"fmt"
	"math"
	"math/big"
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
	rv := reflect.ValueOf(v)
	if rv.CanInt() || rv.CanUint() || rv.CanFloat() {
		return nil
	}
	return fmt.Errorf("expected type float, but value is %#v", v)
}

func validateInteger(v any) error {
	rv := reflect.ValueOf(v)
	if rv.CanInt() || rv.CanUint() {
		return nil
	}
	if rv.CanFloat() {
		f := rv.Float()
		if f == math.Trunc(f) && !math.IsInf(f, 0) {
			return nil
		}
	}
	return fmt.Errorf("expected type integer, but value is %#v", v)
}

// exactNumber returns v as an exact big.Float if v holds a Go integer or a
// non-NaN float. Comparing big.Float values avoids float64 rounding, which
// cannot tell 2^53 from 2^53+1 or MaxInt64 from 2^63.
func exactNumber(v any) (*big.Float, bool) {
	rv := reflect.ValueOf(v)
	switch {
	case rv.CanInt():
		return new(big.Float).SetInt64(rv.Int()), true
	case rv.CanUint():
		return new(big.Float).SetUint64(rv.Uint()), true
	case rv.CanFloat() && !math.IsNaN(rv.Float()):
		return big.NewFloat(rv.Float()), true
	}
	return nil, false
}

// valuesEqual compares two values for const and enum checks. Numbers compare by
// value, so a const of 3 read from the schema (float64) matches 3 from a prompt (int64).
func valuesEqual(a, b any) bool {
	na, okA := exactNumber(a)
	nb, okB := exactNumber(b)
	if okA && okB {
		return na.Cmp(nb) == 0
	}
	// DeepEqual, unlike ==, does not panic on array and object values.
	return reflect.DeepEqual(a, b)
}

var validateTypeFuncs map[Type]validateTypeFunc = map[Type]validateTypeFunc{
	StringType:  validateString,
	BooleanType: validateBoolean,
	IntegerType: validateInteger,
	NumberType:  validateNumber,
}
