package structdiff

import (
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/structtag"
)

// IsEqual compares two Go structs and returns true if they are equal.
// It uses the same comparison logic as GetStructDiff but is more efficient
// as it short-circuits on the first difference found.
// Respects ForceSendFields if present.
// Types of a and b must match exactly, otherwise returns false.
// Note, reflect.DeepEqual() does not work for SDK structs, because ForceSendFields can contain different sets for the same value.
func IsEqual(a, b any) bool {
	v1 := reflect.ValueOf(a)
	v2 := reflect.ValueOf(b)

	if !v1.IsValid() && !v2.IsValid() {
		return true
	}

	if !v1.IsValid() || !v2.IsValid() {
		return false
	}

	if v1.Type() != v2.Type() {
		return false
	}

	return equalValues(v1, v2)
}

// equalValues returns true if v1 and v2 are equal.
func equalValues(v1, v2 reflect.Value) bool {
	if !v1.IsValid() {
		return !v2.IsValid()
	} else if !v2.IsValid() {
		return false
	}

	v1Type := v1.Type()

	if v1Type != v2.Type() {
		return false
	}

	kind := v1.Kind()

	// Perform nil checks for nilable types.
	switch kind {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		v1Nil := v1.IsNil()
		v2Nil := v2.IsNil()
		if v1Nil && v2Nil {
			return true
		}
		if v1Nil || v2Nil {
			return false
		}
	default:
		// Not a nilable type.
		// Proceed with direct comparison below.
	}

	switch kind {
	case reflect.Pointer:
		return equalValues(v1.Elem(), v2.Elem())
	case reflect.Struct:
		return equalStruct(v1, v2)
	case reflect.Slice, reflect.Array:
		if v1.Len() != v2.Len() {
			return false
		}
		for i := range v1.Len() {
			if !equalValues(v1.Index(i), v2.Index(i)) {
				return false
			}
		}
	case reflect.Map:
		if v1Type.Key().Kind() == reflect.String {
			return equalMapStringKey(v1, v2)
		}
		return reflect.DeepEqual(v1.Interface(), v2.Interface())
	default:
		return reflect.DeepEqual(v1.Interface(), v2.Interface())
	}
	return true
}

func equalStruct(s1, s2 reflect.Value) bool {
	t := s1.Type()
	forced1 := getForceSendFields(s1)
	forced2 := getForceSendFields(s2)

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Name == "ForceSendFields" {
			continue
		}

		// Continue traversing embedded structs.
		if sf.Anonymous {
			if !equalValues(s1.Field(i), s2.Field(i)) {
				return false
			}
			continue
		}

		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))

		v1Field := s1.Field(i)
		v2Field := s2.Field(i)

		zero1 := v1Field.IsZero()
		zero2 := v2Field.IsZero()

		if zero1 || zero2 {
			if jsonTag.OmitEmpty() {
				if zero1 {
					if !slices.Contains(forced1, sf.Name) {
						v1Field = reflect.ValueOf(nil)
					}
				}
				if zero2 {
					if !slices.Contains(forced2, sf.Name) {
						v2Field = reflect.ValueOf(nil)
					}
				}
			}
		}

		if !equalValues(v1Field, v2Field) {
			return false
		}
	}
	return true
}

func equalMapStringKey(m1, m2 reflect.Value) bool {
	keySet := map[string]reflect.Value{}
	for _, k := range m1.MapKeys() {
		// Key is always string at this point
		ks := k.Interface().(string)
		keySet[ks] = k
	}
	for _, k := range m2.MapKeys() {
		ks := k.Interface().(string)
		keySet[ks] = k
	}

	for _, k := range keySet {
		v1 := m1.MapIndex(k)
		v2 := m2.MapIndex(k)
		if !equalValues(v1, v2) {
			return false
		}
	}
	return true
}
