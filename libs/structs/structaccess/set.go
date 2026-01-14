package structaccess

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// Set sets the value at the given path inside the target object.
// The target must be a pointer to the object to modify.
func Set(target any, path *structpath.PathNode, value any) error {
	if path.IsRoot() {
		return errors.New("cannot set root value")
	}

	pathLen := path.Len()
	if pathLen == 0 {
		return errors.New("empty path")
	}

	// Validate that target is a pointer
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Pointer {
		return errors.New("target must be a pointer")
	}

	// For single-level paths, get the target directly
	if pathLen == 1 {
		return setValueAtNode(targetVal.Elem(), path, value)
	}

	// For multi-level paths, get the parent container
	parent := path.Parent()
	if parent == nil {
		return errors.New("failed to get parent path")
	}

	// Get the parent container using getValue, passing the original target
	parentVal, err := getValue(target, parent)
	if err != nil {
		return fmt.Errorf("failed to navigate to parent %s: %w", parent.String(), err)
	}

	// Set the value at the final node
	return setValueAtNode(parentVal, path, value)
}

// SetByString sets the value at the given path string inside the target object.
// This is a convenience function that parses the path string and calls Set.
func SetByString(target any, path string, value any) error {
	if path == "" {
		return errors.New("cannot set empty path")
	}

	pathNode, err := structpath.Parse(path)
	if err != nil {
		return err
	}

	return Set(target, pathNode, value)
}

// setValueAtNode sets the value at the specific node in the parent object
func setValueAtNode(parentVal reflect.Value, node *structpath.PathNode, value any) error {
	// Dereference parent if it's a pointer
	for parentVal.Kind() == reflect.Pointer {
		if parentVal.IsNil() {
			return errors.New("parent is nil pointer")
		}
		parentVal = parentVal.Elem()
	}

	valueVal := reflect.ValueOf(value)

	if idx, isIndex := node.Index(); isIndex {
		return setArrayElement(parentVal, idx, valueVal)
	}

	if node.DotStar() || node.BracketStar() {
		return errors.New("wildcards not supported")
	}

	if key, matchValue, isKeyValue := node.KeyValue(); isKeyValue {
		return fmt.Errorf("cannot set value at key-value selector [%s='%s'] - key-value syntax can only be used for path traversal, not as a final target", key, matchValue)
	}

	if key, hasKey := node.StringKey(); hasKey {
		return setFieldOrMapValue(parentVal, key, valueVal)
	}

	return errors.New("unsupported path node type")
}

// setArrayElement sets an element in an array or slice
func setArrayElement(parentVal reflect.Value, index int, valueVal reflect.Value) error {
	kind := parentVal.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return fmt.Errorf("cannot index %s", kind)
	}

	if index < 0 || index >= parentVal.Len() {
		return fmt.Errorf("index %d out of range, length is %d", index, parentVal.Len())
	}

	elemVal := parentVal.Index(index)
	return assignValue(elemVal, valueVal)
}

// setFieldOrMapValue sets a field in a struct or a value in a map, allowing flexible syntax
func setFieldOrMapValue(parentVal reflect.Value, key string, valueVal reflect.Value) error {
	switch parentVal.Kind() {
	case reflect.Struct:
		return setStructField(parentVal, key, valueVal)
	case reflect.Map:
		return setMapValue(parentVal, key, valueVal)
	default:
		return fmt.Errorf("cannot set key %q on %s", key, parentVal.Kind())
	}
}

// setStructField sets a field in a struct and handles ForceSendFields
func setStructField(parentVal reflect.Value, fieldName string, valueVal reflect.Value) error {
	fv, sf, embeddedIndex, ok := findStructFieldByKey(parentVal, fieldName)
	if !ok {
		return fmt.Errorf("field %q not found in %s", fieldName, parentVal.Type())
	}

	// Check if field is settable
	if !fv.CanSet() {
		return fmt.Errorf("field %q cannot be set", sf.Name)
	}

	// Handle ForceSendFields: remove if setting nil, add if setting empty value
	err := updateForceSendFields(parentVal, sf.Name, embeddedIndex, valueVal, sf)
	if err != nil {
		return err
	}

	return assignValue(fv, valueVal)
}

// setMapValue sets a value in a map
func setMapValue(parentVal reflect.Value, key string, valueVal reflect.Value) error {
	kt := parentVal.Type().Key()
	if kt.Kind() != reflect.String {
		return fmt.Errorf("map key must be string, got %s", kt)
	}

	mk := reflect.ValueOf(key)
	if kt != mk.Type() {
		mk = mk.Convert(kt)
	}

	// For maps, we need to handle the value type
	vt := parentVal.Type().Elem()
	convertedValue, err := convertValue(valueVal, vt)
	if err != nil {
		return fmt.Errorf("cannot convert value for map key %q: %w", key, err)
	}

	parentVal.SetMapIndex(mk, convertedValue)
	return nil
}

// assignValue assigns valueVal to targetVal with type compatibility checking
func assignValue(targetVal, valueVal reflect.Value) error {
	if !targetVal.CanSet() {
		return errors.New("target cannot be set")
	}

	convertedValue, err := convertValue(valueVal, targetVal.Type())
	if err != nil {
		return err
	}

	targetVal.Set(convertedValue)
	return nil
}

// convertValue converts valueVal to targetType with compatibility checking
func convertValue(valueVal reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	if !valueVal.IsValid() {
		// Handle nil values - return zero value for the target type
		return reflect.Zero(targetType), nil
	}

	valueType := valueVal.Type()

	// Handle scalar-to-string conversions first (before Go's built-in convertibility).
	// This is critical because Go's built-in ConvertibleTo/Convert has different behavior:
	// - Integers (int, uint8, etc.) convert to string as character codes: 42 → "*", 200 → "È"
	// - Floats and bools are not convertible to string at all and would error
	// We want semantic string representations instead: 42 → "42", true → "true", 3.14 → "3.14"
	if targetType.Kind() == reflect.String {
		switch valueType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			str := strconv.FormatInt(valueVal.Int(), 10)
			return reflect.ValueOf(str), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			str := strconv.FormatUint(valueVal.Uint(), 10)
			return reflect.ValueOf(str), nil
		case reflect.Float32, reflect.Float64:
			str := strconv.FormatFloat(valueVal.Float(), 'g', -1, 64)
			return reflect.ValueOf(str), nil
		case reflect.Bool:
			str := strconv.FormatBool(valueVal.Bool())
			return reflect.ValueOf(str), nil
		default:
			// handled below
		}
	}

	// Handle string-to-scalar conversions
	if valueType.Kind() == reflect.String {
		str := valueVal.String()
		switch targetType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot parse %q as %s: %w", str, targetType.Kind(), err)
			}
			// Check for overflow after parsing
			converted := reflect.ValueOf(val).Convert(targetType)
			if converted.Int() != val {
				return reflect.Value{}, fmt.Errorf("value %d overflows %s", val, targetType.Kind())
			}
			return converted, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot parse %q as %s: %w", str, targetType.Kind(), err)
			}
			// Check for overflow after parsing
			converted := reflect.ValueOf(val).Convert(targetType)
			if converted.Uint() != val {
				return reflect.Value{}, fmt.Errorf("value %d overflows %s", val, targetType.Kind())
			}
			return converted, nil
		case reflect.Float32, reflect.Float64:
			val, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot parse %q as %s: %w", str, targetType.Kind(), err)
			}
			converted := reflect.ValueOf(val).Convert(targetType)
			return converted, nil
		case reflect.Bool:
			val, err := strconv.ParseBool(str)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot parse %q as bool: %w", str, err)
			}
			return reflect.ValueOf(val), nil
		default:
			// handled below
		}
	}

	// Direct assignability check
	if valueType.AssignableTo(targetType) {
		return valueVal, nil
	}

	// Convertibility check (handles typedefed types)
	if valueType.ConvertibleTo(targetType) {
		return valueVal.Convert(targetType), nil
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Pointer {
		elemType := targetType.Elem()
		if valueType.AssignableTo(elemType) {
			// Create a new pointer and set the value
			ptr := reflect.New(elemType)
			ptr.Elem().Set(valueVal)
			return ptr, nil
		}
		if valueType.ConvertibleTo(elemType) {
			ptr := reflect.New(elemType)
			ptr.Elem().Set(valueVal.Convert(elemType))
			return ptr, nil
		}
	}

	// Handle case where value is a pointer but target is not
	if valueType.Kind() == reflect.Pointer && !valueVal.IsNil() {
		elemVal := valueVal.Elem()
		return convertValue(elemVal, targetType)
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", valueType, targetType)
}

// updateForceSendFields handles ForceSendFields when setting values:
// - If setting nil: remove field from ForceSendFields
// - If setting empty value: add field to ForceSendFields (if not already present)
// Only applies to fields with omitempty tag
func updateForceSendFields(parentVal reflect.Value, fieldName string, embeddedIndex int, valueVal reflect.Value, structField reflect.StructField) error {
	isSettingNil := !valueVal.IsValid()
	isSettingEmptyValue := valueVal.IsValid() && isEmptyForOmitEmpty(valueVal)

	// Early return if we don't need to modify ForceSendFields
	if !isSettingNil && !isSettingEmptyValue {
		return nil
	}

	// Check if field has omitempty tag - only omitempty fields need ForceSendFields management
	jsonTag := structtag.JSONTag(structField.Tag.Get("json"))
	if !jsonTag.OmitEmpty() {
		// Non-omitempty fields don't need ForceSendFields management
		return nil
	}

	// Find the appropriate ForceSendFields slice to modify
	forceSendFieldsSlice := findForceSendFieldsForSetting(parentVal, embeddedIndex)
	if !forceSendFieldsSlice.IsValid() {
		// No ForceSendFields to update
		return nil
	}

	if isSettingNil {
		// Remove from ForceSendFields
		removeFromForceSendFields(forceSendFieldsSlice, fieldName)
	} else if isSettingEmptyValue {
		// Add to ForceSendFields if not already present
		addToForceSendFields(forceSendFieldsSlice, fieldName)
	}

	return nil
}

// findForceSendFieldsForSetting finds the correct ForceSendFields slice to modify
// This should match the logic in get.go's getForceSendFieldsForFromTyped
// Only the struct that contains the ForceSendFields can manage its own fields
// embeddedIndex: -1 for direct fields, or the index of the embedded struct
func findForceSendFieldsForSetting(parentVal reflect.Value, embeddedIndex int) reflect.Value {
	if embeddedIndex == -1 {
		// Direct field - check if parent struct has its own ForceSendFields
		// We need to check the struct type directly, not through field promotion
		parentType := parentVal.Type()
		for i := range parentType.NumField() {
			field := parentType.Field(i)
			if field.Name == "ForceSendFields" && !field.Anonymous {
				// Parent has direct ForceSendFields
				return parentVal.Field(i)
			}
		}
		// Parent struct has no direct ForceSendFields, so no management possible
		return reflect.Value{}
	} else {
		// Embedded field - look for ForceSendFields in the embedded struct
		embeddedField := parentVal.Field(embeddedIndex)
		embeddedStruct := getEmbeddedStructForSetting(embeddedField)
		if !embeddedStruct.IsValid() {
			return reflect.Value{}
		}
		fsf := embeddedStruct.FieldByName("ForceSendFields")
		if fsf.IsValid() {
			return fsf
		}
		// Embedded struct has no ForceSendFields, so no management possible
		return reflect.Value{}
	}
}

// getEmbeddedStructForSetting gets the embedded struct for setting operations
// Creates nil pointers if needed
func getEmbeddedStructForSetting(fieldValue reflect.Value) reflect.Value {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			// Create new instance if needed
			if fieldValue.CanSet() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			} else {
				return reflect.Value{}
			}
		}
		fieldValue = fieldValue.Elem()
	}
	if fieldValue.Kind() == reflect.Struct {
		return fieldValue
	}
	return reflect.Value{}
}

// removeFromForceSendFields removes fieldName from the ForceSendFields slice
func removeFromForceSendFields(forceSendFieldsSlice reflect.Value, fieldName string) {
	// Get the original []string slice
	fields := forceSendFieldsSlice.Interface().([]string)

	// Find the index of the field to remove
	index := slices.Index(fields, fieldName)
	if index == -1 {
		return // Field not found, nothing to remove
	}

	// Remove the field using slices.Delete
	newFields := slices.Delete(fields, index, index+1)
	forceSendFieldsSlice.Set(reflect.ValueOf(newFields))
}

// addToForceSendFields adds fieldName to the ForceSendFields slice if not already present
func addToForceSendFields(forceSendFieldsSlice reflect.Value, fieldName string) {
	// Get the original []string slice
	fields := forceSendFieldsSlice.Interface().([]string)

	// Check if already present
	if slices.Contains(fields, fieldName) {
		return // Already present
	}

	// Add the new field
	newFields := append(fields, fieldName)
	forceSendFieldsSlice.Set(reflect.ValueOf(newFields))
}
