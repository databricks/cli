package structdiff

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
)

type Change struct {
	Field string
	Old   any
	New   any
}

// GetStructDiff compares two Go structs and returns a list of Changes or an error.
// Respects ForceSendFields if present.
// Types of and ab must match exactly, otherwise returns an error.
func GetStructDiff(a, b any) ([]Change, error) {
	v1 := reflect.ValueOf(a)
	v2 := reflect.ValueOf(b)

	if !v1.IsValid() && !v2.IsValid() {
		return nil, nil
	}

	var changes []Change

	if !v1.IsValid() || !v2.IsValid() {
		add("", v1, v2, &changes)
		return changes, nil
	}

	if v1.Type() != v2.Type() {
		return nil, fmt.Errorf("type mismatch: %v vs %v", v1.Type(), v2.Type())
	}

	diffValues("", v1, v2, &changes)
	return changes, nil
}

// diffValues appends changes between v1 and v2 to the slice.  path is the current
// JSON-style path (dot + brackets).  At the root path is "".
func diffValues(path string, v1, v2 reflect.Value, changes *[]Change) {
	// both invalid – nothing to compare
	if !v1.IsValid() && !v2.IsValid() {
		return
	}

	// Promote invalid to nil interface so valueOf can handle it uniformly
	if !v1.IsValid() {
		v1 = reflect.ValueOf(nil)
	}
	if !v2.IsValid() {
		v2 = reflect.ValueOf(nil)
	}

	// Both nil (or typed-nil) – identical
	if isNil(v1) && isNil(v2) {
		return
	}

	// This should not happen; if it does, record this a full change
	if v1.Type() != v2.Type() {
		add(path, v1, v2, changes)
		return
	}

	// Exactly one nil (interface, slice, pointer, etc.)
	if isNil(v1) || isNil(v2) {
		add(path, v1, v2, changes)
		return
	}

	// From here on, both non-nil and same type
	kind := v1.Kind()

	switch kind {
	case reflect.Pointer:
		diffValues(path, v1.Elem(), v2.Elem(), changes)
	case reflect.Struct:
		diffStruct(path, v1, v2, changes)
	case reflect.Slice, reflect.Array:
		if v1.Len() != v2.Len() {
			add(path, v1, v2, changes)
		} else {
			for i := range v1.Len() {
				idxPath := fmt.Sprintf("%s[%d]", path, i)
				diffValues(idxPath, v1.Index(i), v2.Index(i), changes)
			}
		}
	case reflect.Map:
		diffMap(path, v1, v2, changes)
	default: // primitives, interfaces, etc.
		if !reflect.DeepEqual(v1.Interface(), v2.Interface()) {
			add(path, v1, v2, changes)
		}
	}
}

func diffStruct(path string, s1, s2 reflect.Value, changes *[]Change) {
	t := s1.Type()
	forced1 := getForceSendFields(s1)
	forced2 := getForceSendFields(s2)

	isForceDifferent := !slices.Equal(forced1, forced2)

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Name == "ForceSendFields" {
			continue // ignore unexported and meta field itself
		}

		fieldPath := join(path, sf.Name)
		v1Field := s1.Field(i)
		v2Field := s2.Field(i)

		if isForceDifferent {
			hasOmitEmpty := strings.Contains(sf.Tag.Get("json"), "omitempty")

			// Special handling when both values are zero but ForceSendFields differ.
			if hasOmitEmpty && v1Field.IsZero() && v2Field.IsZero() {
				f1 := slices.Contains(forced1, sf.Name)
				f2 := slices.Contains(forced2, sf.Name)
				if f1 != f2 {
					oldI := any(nil)
					newI := any(nil)
					if f1 { // first struct forces send – explicit empty value
						oldI = v1Field.Interface()
					}
					if f2 {
						newI = v2Field.Interface()
					}
					*changes = append(*changes, Change{Field: fieldPath, Old: oldI, New: newI})
					continue // no deeper diff needed – we already recorded it
				}
			}
		}

		diffValues(fieldPath, v1Field, v2Field, changes)
	}
}

func diffMap(path string, m1, m2 reflect.Value, changes *[]Change) {
	if m1.Kind() != reflect.Map || m2.Kind() != reflect.Map {
		add(path, m1, m2, changes)
		return
	}

	keySet := map[string]reflect.Value{}
	for _, k := range m1.MapKeys() {
		keySet[keyToString(k)] = k
	}
	for _, k := range m2.MapKeys() {
		keySet[keyToString(k)] = k
	}

	var keys []string
	for s := range keySet {
		keys = append(keys, s)
	}
	sort.Strings(keys)

	for _, ks := range keys {
		k := keySet[ks]
		v1 := m1.MapIndex(k)
		v2 := m2.MapIndex(k)
		kp := fmt.Sprintf("%s[%#v]", path, ks)
		diffValues(kp, v1, v2, changes)
	}
}

func getForceSendFields(v reflect.Value) []string {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}
	fsField := v.FieldByName("ForceSendFields")
	if !fsField.IsValid() || fsField.Kind() != reflect.Slice {
		return nil
	}
	result, ok := fsField.Interface().([]string)
	if ok {
		return result
	}
	return nil
}

func keyToString(k reflect.Value) string { return fmt.Sprint(k.Interface()) }

func join(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func add(path string, v1, v2 reflect.Value, changes *[]Change) {
	*changes = append(*changes, Change{Field: path, Old: valueOf(v1), New: valueOf(v2)})
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func valueOf(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if isNil(v) {
		return reflect.Zero(v.Type()).Interface()
	}
	return v.Interface()
}
