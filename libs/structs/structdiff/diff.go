package structdiff

import (
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/registry"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

type Change struct {
	Path *structpath.PathNode
	Old  any
	New  any
}

// GetStructDiff compares two Go structs and returns a list of Changes or an error.
// Respects ForceSendFields if present.
// Types of a and b must match exactly, otherwise returns an error.
//
// A slice whose element type is registered as a keyed slice (see libs/structs/registry)
// is matched by key value instead of by index. The element is addressed in the path as
// [='value'] — the key field is omitted, because the value identifies the element.
// A diff on a matched element's own key field is therefore not a real change and is
// dropped, which lets the same identity carried under a different field (e.g. user_name
// vs service_principal_name for one principal) compare equal.
func GetStructDiff(a, b any) ([]Change, error) {
	v1 := reflect.ValueOf(a)
	v2 := reflect.ValueOf(b)

	if !v1.IsValid() && !v2.IsValid() {
		return nil, nil
	}

	var changes []Change

	if !v1.IsValid() || !v2.IsValid() {
		changes = append(changes, Change{Path: nil, Old: v1.Interface(), New: v2.Interface()})
		return changes, nil
	}

	if v1.Type() != v2.Type() {
		return nil, fmt.Errorf("type mismatch: %v vs %v", v1.Type(), v2.Type())
	}

	if err := diffValues(nil, v1, v2, &changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// diffValues appends changes between v1 and v2 to the slice.  path is the current
// JSON-style path (dot + brackets).  At the root path is "".
func diffValues(path *structpath.PathNode, v1, v2 reflect.Value, changes *[]Change) error {
	if !v1.IsValid() {
		if !v2.IsValid() {
			return nil
		}
		// Dereference non-nil pointers for consistency with the both-non-nil case,
		// where pointers are recursively dereferenced via "case reflect.Pointer".
		for v2.Kind() == reflect.Pointer && !v2.IsNil() {
			v2 = v2.Elem()
		}
		*changes = append(*changes, Change{Path: path, Old: nil, New: v2.Interface()})
		return nil
	} else if !v2.IsValid() {
		// v1 is valid
		for v1.Kind() == reflect.Pointer && !v1.IsNil() {
			v1 = v1.Elem()
		}
		*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: nil})
		return nil
	}

	v1Type := v1.Type()

	// This should not happen; if it does, record this a full change
	if v1Type != v2.Type() {
		*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
		return nil
	}

	if IsOpaqueStruct(v1Type) {
		equal, err := equalJSON(v1, v2)
		if err != nil {
			return err
		}
		if !equal {
			*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
		}
		return nil
	}

	kind := v1.Kind()

	// Perform nil checks for nilable types.
	switch kind {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		v1Nil := v1.IsNil()
		v2Nil := v2.IsNil()
		if v1Nil && v2Nil {
			return nil
		}
		if v1Nil || v2Nil {
			*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
			return nil
		}
	default:
		// Not a nilable type.
		// Proceed with direct comparison below.
	}

	switch kind {
	case reflect.Pointer:
		return diffValues(path, v1.Elem(), v2.Elem(), changes)
	case reflect.Struct:
		return diffStruct(path, v1, v2, changes)
	case reflect.Slice, reflect.Array:
		if keyFields := registry.KeyFields(v1Type.Elem()); keyFields != nil {
			return diffSliceByKey(path, v1, v2, keyFields, changes)
		} else if v1.Len() != v2.Len() {
			*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
		} else {
			for i := range v1.Len() {
				node := structpath.NewIndex(path, i)
				if err := diffValues(node, v1.Index(i), v2.Index(i), changes); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		if v1Type.Key().Kind() == reflect.String {
			return diffMapStringKey(path, v1, v2, changes)
		} else {
			deepEqualValues(path, v1, v2, changes)
		}
	default:
		deepEqualValues(path, v1, v2, changes)
	}
	return nil
}

func deepEqualValues(path *structpath.PathNode, v1, v2 reflect.Value, changes *[]Change) {
	if !reflect.DeepEqual(v1.Interface(), v2.Interface()) {
		*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
	}
}

func diffStruct(path *structpath.PathNode, s1, s2 reflect.Value, changes *[]Change) error {
	t := s1.Type()
	forced1 := getForceSendFields(s1)
	forced2 := getForceSendFields(s2)

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Name == "ForceSendFields" {
			continue
		}

		// Continue traversing embedded structs. Do not add the key to the path though. An
		// anonymous field carrying a json name is not one of these: encoding/json serializes it
		// as a nested object, so it is handled as a named field below.
		if structaccess.IsFlattenedEmbed(sf) {
			if err := diffValues(path, s1.Field(i), s2.Field(i), changes); err != nil {
				return err
			}
			continue
		}

		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))

		// Resolve field name from JSON tag or fall back to Go field name
		fieldName := jsonTag.Name()
		if fieldName == "-" {
			continue
		}

		isEmbed := sf.Name == structaccess.EmbeddedSliceFieldName

		if fieldName == "" || isEmbed {
			fieldName = sf.Name
		}

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

		// EmbeddedSlice: diff at parent path level without adding field name.
		var node *structpath.PathNode
		if isEmbed {
			node = path
		} else {
			node = structpath.NewDotString(path, fieldName)
		}

		if err := diffValues(node, v1Field, v2Field, changes); err != nil {
			return err
		}
	}
	return nil
}

func diffMapStringKey(path *structpath.PathNode, m1, m2 reflect.Value, changes *[]Change) error {
	keySet := map[string]reflect.Value{}
	for _, k := range m1.MapKeys() {
		// Caller guarantees the key kind is String; use Value.String() rather
		// than a .(string) assertion, which panics on a named string key type
		// (e.g. `type ScriptHook string`) whose dynamic type is not string.
		keySet[k.String()] = k
	}
	for _, k := range m2.MapKeys() {
		keySet[k.String()] = k
	}

	keys := slices.Sorted(maps.Keys(keySet))

	for _, ks := range keys {
		k := keySet[ks]
		v1 := m1.MapIndex(k)
		v2 := m2.MapIndex(k)
		node := structpath.NewBracketString(path, ks)
		if err := diffValues(node, v1, v2, changes); err != nil {
			return err
		}
	}
	return nil
}

func getForceSendFields(v reflect.Value) []string {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}
	fsField := v.FieldByName("ForceSendFields")
	if !fsField.IsValid() || fsField.Kind() != reflect.Slice {
		return nil
	}
	result, ok := reflect.TypeAssert[[]string](fsField)
	if ok {
		return result
	}
	return nil
}

// appendSkippingKeyFields appends pairChanges to changes, dropping any change on an
// element's own key field: a direct child of node whose field name is one of
// keyFields. Such a diff is not a real change — the element is matched by key value,
// so a key-field difference only means the same identity is carried under a different
// field (e.g. service_principal_name vs user_name).
func appendSkippingKeyFields(changes *[]Change, pairChanges []Change, node *structpath.PathNode, keyFields []string) {
	for _, ch := range pairChanges {
		if ch.Path.Parent() == node {
			if field, ok := ch.Path.StringKey(); ok && slices.Contains(keyFields, field) {
				continue
			}
		}
		*changes = append(*changes, ch)
	}
}

// diffSliceByKey compares two slices whose element type is a registered keyed slice.
// Elements are matched by their key value (registry.ElementKey) instead of by index,
// and addressed as [='value']. Duplicate keys are allowed and matched in order.
func diffSliceByKey(path *structpath.PathNode, v1, v2 reflect.Value, keyFields []string, changes *[]Change) error {
	// Build lists of elements grouped by key value, preserving order within each key.
	elements1 := make(map[string][]reflect.Value)
	elements2 := make(map[string][]reflect.Value)
	seen := make(map[string]bool)
	var orderedKeys []string

	group := func(v reflect.Value, into map[string][]reflect.Value) {
		for i := range v.Len() {
			elem := v.Index(i)
			key, _ := structaccess.ElementKeyValue(elem, keyFields)
			into[key] = append(into[key], elem)
			if !seen[key] {
				seen[key] = true
				orderedKeys = append(orderedKeys, key)
			}
		}
	}
	group(v1, elements1)
	group(v2, elements2)

	for _, key := range orderedKeys {
		list1 := elements1[key]
		list2 := elements2[key]

		// The element is addressed by value only ([='value']); the key field is omitted,
		// so both sides render identically even when they carry the same identity under a
		// different field. Resolution back to an element is the type-aware resolver's job
		// (structaccess, via the registry).
		node := structpath.NewKeyValue(path, "", key)

		// Match elements in order.
		minLen := min(len(list1), len(list2))
		for i := range minLen {
			var pairChanges []Change
			if err := diffValues(node, list1[i], list2[i], &pairChanges); err != nil {
				return err
			}
			appendSkippingKeyFields(changes, pairChanges, node, keyFields)
		}

		// Handle extra elements in old (deleted)
		for i := minLen; i < len(list1); i++ {
			*changes = append(*changes, Change{Path: node, Old: list1[i].Interface(), New: nil})
		}

		// Handle extra elements in new (added)
		for i := minLen; i < len(list2); i++ {
			*changes = append(*changes, Change{Path: node, Old: nil, New: list2[i].Interface()})
		}
	}
	return nil
}
