package main

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// isEnumType checks if a type is an enum (string type with a Values() method)
func isEnumType(typ reflect.Type) bool {
	// Must be a string type
	if typ.Kind() != reflect.String {
		return false
	}

	// Check for Values() method on both the type and its pointer type
	var valuesMethod reflect.Method
	var hasValues bool

	// First check on the type itself
	valuesMethod, hasValues = typ.MethodByName("Values")
	if !hasValues {
		// Then check on the pointer type
		ptrType := reflect.PointerTo(typ)
		valuesMethod, hasValues = ptrType.MethodByName("Values")
	}

	if !hasValues {
		return false
	}

	// Check method signature: func (T) Values() []T or func (*T) Values() []T
	methodType := valuesMethod.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
		return false
	}

	// Check return type is slice of the enum type
	returnType := methodType.Out(0)
	if returnType.Kind() != reflect.Slice {
		return false
	}

	elementType := returnType.Elem()
	return elementType == typ
}

// getEnumValues extracts enum values by calling the Values() method
func getEnumValues(typ reflect.Type) ([]string, error) {
	if !isEnumType(typ) {
		return nil, fmt.Errorf("type %s is not an enum type", typ.Name())
	}

	// Create a zero value of the type
	enumValue := reflect.Zero(typ)

	// Check if the method is on the type or pointer type
	valuesMethod := enumValue.MethodByName("Values")
	if !valuesMethod.IsValid() {
		// Try on pointer type
		enumPtr := reflect.New(typ)
		enumPtr.Elem().Set(enumValue)
		valuesMethod = enumPtr.MethodByName("Values")
		if !valuesMethod.IsValid() {
			return nil, fmt.Errorf("Values() method not found on type %s", typ.Name())
		}
	}

	results := valuesMethod.Call(nil)
	if len(results) != 1 {
		return nil, errors.New("Values() method should return exactly one value")
	}

	valuesSlice := results[0]
	if valuesSlice.Kind() != reflect.Slice {
		return nil, errors.New("Values() method should return a slice")
	}

	// Extract string values from the slice
	var enumStrings []string
	for i := range valuesSlice.Len() {
		value := valuesSlice.Index(i)
		enumStrings = append(enumStrings, value.String())
	}

	return enumStrings, nil
}

// extractEnumFields walks through a struct type and returns the valid values by the
// pattern of the enum fields.
func extractEnumFields(typ reflect.Type) (map[string][]string, error) {
	fieldsByPattern := make(map[string][]string)

	err := structwalk.WalkType(typ, func(path *structpath.PatternNode, fieldType reflect.Type, field *reflect.StructField) bool {
		if path == nil {
			return true
		}

		// Do not generate enum validation code for fields that are internal or readonly.
		if field != nil {
			bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
			if bundleTag.Internal() || bundleTag.ReadOnly() {
				return false
			}
		}

		// Check if this field type is an enum
		if isEnumType(fieldType) {
			// Get enum values
			enumValues, err := getEnumValues(fieldType)
			if err != nil {
				// Skip this field if we can't get enum values
				return true
			}

			fieldsByPattern[path.String()] = enumValues
		}
		return true
	})

	return fieldsByPattern, err
}
