package tnresources

import (
	"reflect"
)

// filterForceSendFields creates a new slice with fields present only in the provided type.
// We must use that when copying structs because JSON marshaller in SDK crashes if it sees unknown field.
func filterForceSendFields[T any](fields []string) []string {
	var result []string
	typeOfT := reflect.TypeOf((*T)(nil)).Elem()

	for _, field := range fields {
		if _, ok := typeOfT.FieldByName(field); ok {
			result = append(result, field)
		}
	}

	return result
}
