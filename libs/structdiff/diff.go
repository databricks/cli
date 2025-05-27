package structdiff

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type Change struct {
	Field string
	Old   any
	New   any
}

type pathNode struct {
	Prev *pathNode
	Key  string
	// If Index >= 0, the node specifies a slice/array index in Index.
	// If Index == -1, the node specifies a struct attribute in Key
	// If Index == -2, the node specifies a map key in Key
	Index int
}

func (p *pathNode) String() string {
	if p == nil {
		return ""
	}
	if p.Index >= 0 {
		return p.Prev.String() + "[" + strconv.Itoa(p.Index) + "]"
	}
	if p.Index == -1 {
		return p.Prev.String() + "." + p.Key
	}
	return fmt.Sprintf("%s[%q]", p.Prev.String(), p.Key)
}

// GetStructDiff compares two Go structs and returns a list of Changes or an error.
// Respects ForceSendFields if present.
// Types of a and b must match exactly, otherwise returns an error.
func GetStructDiff(a, b any) ([]Change, error) {
	v1 := reflect.ValueOf(a)
	v2 := reflect.ValueOf(b)

	if !v1.IsValid() && !v2.IsValid() {
		return nil, nil
	}

	var changes []Change

	if !v1.IsValid() || !v2.IsValid() {
		changes = append(changes, Change{Field: "", Old: v1.Interface(), New: v2.Interface()})
		return changes, nil
	}

	if v1.Type() != v2.Type() {
		return nil, fmt.Errorf("type mismatch: %v vs %v", v1.Type(), v2.Type())
	}

	diffValues(nil, v1, v2, &changes)
	return changes, nil
}

// diffValues appends changes between v1 and v2 to the slice.  path is the current
// JSON-style path (dot + brackets).  At the root path is "".
func diffValues(path *pathNode, v1, v2 reflect.Value, changes *[]Change) {
	if !v1.IsValid() {
		if !v2.IsValid() {
			return
		}

		*changes = append(*changes, Change{Field: path.String(), Old: nil, New: v2.Interface()})
		return
	} else if !v2.IsValid() {
		// v1 is valid
		*changes = append(*changes, Change{Field: path.String(), Old: v1.Interface(), New: nil})
		return
	}

	v1Type := v1.Type()

	// This should not happen; if it does, record this a full change
	if v1Type != v2.Type() {
		*changes = append(*changes, Change{Field: path.String(), Old: v1.Interface(), New: v2.Interface()})
		return
	}

	kind := v1.Kind()

	switch kind {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		v1Nil := v1.IsNil()
		v2Nil := v2.IsNil()
		if v1Nil && v2Nil {
			return
		}
		if v1Nil || v2Nil {
			*changes = append(*changes, Change{Field: path.String(), Old: v1.Interface(), New: v2.Interface()})
			return
		}
	}

	switch kind {
	case reflect.Pointer:
		diffValues(path, v1.Elem(), v2.Elem(), changes)
	case reflect.Struct:
		diffStruct(path, v1, v2, changes)
	case reflect.Slice, reflect.Array:
		if v1.Len() != v2.Len() {
			*changes = append(*changes, Change{Field: path.String(), Old: v1.Interface(), New: v2.Interface()})
		} else {
			for i := range v1.Len() {
				node := pathNode{Prev: path, Index: i}
				diffValues(&node, v1.Index(i), v2.Index(i), changes)
			}
		}
	case reflect.Map:
		if v1Type.Key().Kind() == reflect.String {
			diffMapStringKey(path, v1, v2, changes)
		} else {
			deepEqualValues(path, v1, v2, changes)
		}
	default:
		deepEqualValues(path, v1, v2, changes)
	}
}

func deepEqualValues(path *pathNode, v1, v2 reflect.Value, changes *[]Change) {
	if !reflect.DeepEqual(v1.Interface(), v2.Interface()) {
		*changes = append(*changes, Change{Field: path.String(), Old: v1.Interface(), New: v2.Interface()})
	}
}

func diffStruct(path *pathNode, s1, s2 reflect.Value, changes *[]Change) {
	t := s1.Type()
	forced1 := getForceSendFields(s1)
	forced2 := getForceSendFields(s2)

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Name == "ForceSendFields" {
			continue
		}

		node := pathNode{Prev: path, Key: sf.Name, Index: -1}
		v1Field := s1.Field(i)
		v2Field := s2.Field(i)

		hasOmitEmpty := strings.Contains(sf.Tag.Get("json"), "omitempty")

		if hasOmitEmpty {
			if v1Field.IsZero() {
				if !slices.Contains(forced1, sf.Name) {
					v1Field = reflect.ValueOf(nil)
				}
			}
			if v2Field.IsZero() {
				if !slices.Contains(forced2, sf.Name) {
					v2Field = reflect.ValueOf(nil)
				}
			}
		}

		diffValues(&node, v1Field, v2Field, changes)
	}
}

func diffMapStringKey(path *pathNode, m1, m2 reflect.Value, changes *[]Change) {
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

	var keys []string
	for s := range keySet {
		keys = append(keys, s)
	}
	sort.Strings(keys)

	for _, ks := range keys {
		k := keySet[ks]
		v1 := m1.MapIndex(k)
		v2 := m2.MapIndex(k)
		node := pathNode{
			Prev:  path,
			Key:   ks,
			Index: -2,
		}
		diffValues(&node, v1, v2, changes)
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
