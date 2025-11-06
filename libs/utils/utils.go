package utils

import (
	"reflect"
	"sort"
)

func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FilterFields creates a new slice with fields present only in the provided type,
// excluding any fields specified in the excludeFields list.
// We must use that when copying structs because JSON marshaller in SDK crashes if it sees unknown field.
func FilterFields[T any](fields []string, excludeFields ...string) []string {
	var result []string
	typeOfT := reflect.TypeOf((*T)(nil)).Elem()

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
