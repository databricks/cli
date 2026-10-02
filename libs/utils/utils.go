package utils

import (
	"reflect"
)

// FilterFields creates a new slice with fields present only in the provided type,
// excluding any fields specified in the excludeFields list.
// We must use that when copying structs because JSON marshaller in SDK crashes if it sees unknown field.
func FilterFields[T any](fields []string, excludeFields ...string) []string {
	return FilterFieldsType(reflect.TypeFor[T](), fields, excludeFields...)
}

// FilterFieldsType is FilterFields with the destination type supplied as a reflect.Type
// rather than a type parameter, for callers that only know the type at runtime.
func FilterFieldsType(typeOfT reflect.Type, fields []string, excludeFields ...string) []string {
	var result []string

	excludeMap := make(map[string]bool)
	for _, exclude := range excludeFields {
		excludeMap[exclude] = true
	}

	for _, field := range fields {
		// Skip if field is in exclude list
		if excludeMap[field] {
			continue
		}
		if _, ok := typeOfT.FieldByName(field); ok {
			result = append(result, field)
		}
	}

	return result
}
