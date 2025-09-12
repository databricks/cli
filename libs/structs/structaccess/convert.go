package structaccess

import (
	"fmt"
	"reflect"
)

func ConvertToString(value any) (string, error) {
	originalValue := value

	// Handle pointers by dereferencing them first
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "", nil
		}
		value = rv.Elem().Interface()
	}

	// Use the same conversion logic as convertValue for consistency
	valueVal := reflect.ValueOf(value)
	stringType := reflect.TypeOf("")

	convertedValue, err := convertValue(valueVal, stringType)
	if err != nil {
		return "", fmt.Errorf("unsupported type for string conversion: %T", originalValue)
	}

	return convertedValue.String(), nil
}
