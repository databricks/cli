package dresources

import (
	"reflect"

	"github.com/databricks/databricks-sdk-go/retries"
)

// filterFields creates a new slice with fields present only in the provided type,
// excluding any fields specified in the excludeFields list.
// We must use that when copying structs because JSON marshaller in SDK crashes if it sees unknown field.
func filterFields[T any](fields []string, excludeFields ...string) []string {
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

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	e := err.(*retries.Err)
	if e == nil {
		return false
	}
	return !e.Halt
}
