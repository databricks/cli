package jsonschema

import (
	"fmt"
	"reflect"
	"slices"
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

func validateNumber(v any) error {
	if !slices.Contains([]reflect.Kind{reflect.Float32, reflect.Float64},
		reflect.TypeOf(v).Kind()) {
		return fmt.Errorf("expected type float, but value is %#v", v)
	}
	return nil
}

func validateInteger(v any) error {
	if !slices.Contains([]reflect.Kind{
		reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64,
	},
		reflect.TypeOf(v).Kind()) {
		return fmt.Errorf("expected type integer, but value is %#v", v)
	}
	return nil
}

var validateTypeFuncs map[Type]validateTypeFunc = map[Type]validateTypeFunc{
	StringType:  validateString,
	BooleanType: validateBoolean,
	IntegerType: validateInteger,
	NumberType:  validateNumber,
}
