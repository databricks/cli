package template

import (
	"fmt"
	"reflect"

	"golang.org/x/exp/slices"
)

type validator func(v any) error

func validateType(v any, fieldType PropertyType) error {
	validateFunc, ok := validators[fieldType]
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
	if !slices.Contains([]reflect.Kind{reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64},
		reflect.TypeOf(v).Kind()) {
		return fmt.Errorf("expected type float, but value is %#v", v)
	}
	return nil
}

func validateInteger(v any) error {
	if !slices.Contains([]reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64},
		reflect.TypeOf(v).Kind()) {
		return fmt.Errorf("expected type integer, but value is %#v", v)
	}
	return nil
}

var validators map[PropertyType]validator = map[PropertyType]validator{
	PropertyTypeString:  validateString,
	PropertyTypeBoolean: validateBoolean,
	PropertyTypeInt:     validateInteger,
	PropertyTypeNumber:  validateNumber,
}
