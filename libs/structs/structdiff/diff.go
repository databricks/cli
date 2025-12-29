package structdiff

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

type Change struct {
	Path *structpath.PathNode
	Old  any
	New  any
}

// KeyFunc extracts a key field name and value from a slice element.
// It can be either:
//   - func(T) (string, string) - typed function for specific element type T
//   - func(any) (string, string) - generic function accepting any element
//
// The function returns (keyField, keyValue). The keyField is typically a field name
// like "task_key", and keyValue is the value that uniquely identifies the element.
type KeyFunc = any

// keyFuncCaller wraps a KeyFunc and provides a type-checked Call method.
type keyFuncCaller struct {
	fn      reflect.Value
	argType reflect.Type
}

func newKeyFuncCaller(fn any) (*keyFuncCaller, error) {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("KeyFunc must be a function, got %T", fn)
	}
	t := v.Type()
	if t.NumIn() != 1 {
		return nil, fmt.Errorf("KeyFunc must have exactly 1 parameter, got %d", t.NumIn())
	}
	if t.NumOut() != 2 {
		return nil, fmt.Errorf("KeyFunc must return exactly 2 values, got %d", t.NumOut())
	}
	if t.Out(0).Kind() != reflect.String || t.Out(1).Kind() != reflect.String {
		return nil, fmt.Errorf("KeyFunc must return (string, string), got (%v, %v)", t.Out(0), t.Out(1))
	}
	return &keyFuncCaller{fn: v, argType: t.In(0)}, nil
}

func (c *keyFuncCaller) call(elem any) (string, string) {
	elemValue := reflect.ValueOf(elem)
	out := c.fn.Call([]reflect.Value{elemValue})
	keyField := out[0].String()
	keyValue := out[1].String()
	return keyField, keyValue
}

// diffContext holds configuration for the diff operation.
type diffContext struct {
	sliceKeys map[string]KeyFunc
}

// GetStructDiff compares two Go structs and returns a list of Changes or an error.
// Respects ForceSendFields if present.
// Types of a and b must match exactly, otherwise returns an error.
//
// The sliceKeys parameter maps path patterns to functions that extract
// key field/value pairs from slice elements. When provided, slices at matching
// paths are compared as maps keyed by (keyField, keyValue) instead of by index.
// Path patterns use dot notation (e.g., "tasks" or "job.tasks").
// The [*] wildcard matches any slice index in the path.
// Note, key wildcard is not supported yet ("a.*.c")
// Pass nil if no slice key functions are needed.
func GetStructDiff(a, b any, sliceKeys map[string]KeyFunc) ([]Change, error) {
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

	ctx := &diffContext{sliceKeys: sliceKeys}
	if err := diffValues(ctx, nil, v1, v2, &changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// diffValues appends changes between v1 and v2 to the slice.  path is the current
// JSON-style path (dot + brackets).  At the root path is "".
func diffValues(ctx *diffContext, path *structpath.PathNode, v1, v2 reflect.Value, changes *[]Change) error {
	if !v1.IsValid() {
		if !v2.IsValid() {
			return nil
		}

		*changes = append(*changes, Change{Path: path, Old: nil, New: v2.Interface()})
		return nil
	} else if !v2.IsValid() {
		// v1 is valid
		*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: nil})
		return nil
	}

	v1Type := v1.Type()

	// This should not happen; if it does, record this a full change
	if v1Type != v2.Type() {
		*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
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
		return diffValues(ctx, path, v1.Elem(), v2.Elem(), changes)
	case reflect.Struct:
		return diffStruct(ctx, path, v1, v2, changes)
	case reflect.Slice, reflect.Array:
		if keyFunc := ctx.findKeyFunc(path); keyFunc != nil {
			return diffSliceByKey(ctx, path, v1, v2, keyFunc, changes)
		} else if v1.Len() != v2.Len() {
			*changes = append(*changes, Change{Path: path, Old: v1.Interface(), New: v2.Interface()})
		} else {
			for i := range v1.Len() {
				node := structpath.NewIndex(path, i)
				if err := diffValues(ctx, node, v1.Index(i), v2.Index(i), changes); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		if v1Type.Key().Kind() == reflect.String {
			return diffMapStringKey(ctx, path, v1, v2, changes)
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

func diffStruct(ctx *diffContext, path *structpath.PathNode, s1, s2 reflect.Value, changes *[]Change) error {
	t := s1.Type()
	forced1 := getForceSendFields(s1)
	forced2 := getForceSendFields(s2)

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Name == "ForceSendFields" {
			continue
		}

		// Continue traversing embedded structs. Do not add the key to the path though.
		if sf.Anonymous {
			if err := diffValues(ctx, path, s1.Field(i), s2.Field(i), changes); err != nil {
				return err
			}
			continue
		}

		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))

		// Resolve field name from JSON tag or fall back to Go field name
		fieldName := jsonTag.Name()
		if fieldName == "" {
			fieldName = sf.Name
		}
		node := structpath.NewDotString(path, fieldName)

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

		if err := diffValues(ctx, node, v1Field, v2Field, changes); err != nil {
			return err
		}
	}
	return nil
}

func diffMapStringKey(ctx *diffContext, path *structpath.PathNode, m1, m2 reflect.Value, changes *[]Change) error {
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
		node := structpath.NewBracketString(path, ks)
		if err := diffValues(ctx, node, v1, v2, changes); err != nil {
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
	result, ok := fsField.Interface().([]string)
	if ok {
		return result
	}
	return nil
}

// findKeyFunc returns the KeyFunc for the given path, or nil if none matches.
// Path patterns support [*] to match any slice index.
func (ctx *diffContext) findKeyFunc(path *structpath.PathNode) KeyFunc {
	if ctx.sliceKeys == nil {
		return nil
	}
	pathStr := pathToPattern(path)
	return ctx.sliceKeys[pathStr]
}

// pathToPattern converts a PathNode to a pattern string for matching.
// Slice indices are converted to [*] wildcard.
func pathToPattern(path *structpath.PathNode) string {
	if path == nil {
		return ""
	}

	components := path.AsSlice()
	var result strings.Builder

	for i, node := range components {
		if idx, ok := node.Index(); ok {
			// Convert numeric index to wildcard
			_ = idx
			result.WriteString("[*]")
		} else if key, value, ok := node.KeyValue(); ok {
			// Key-value syntax
			result.WriteString("[")
			result.WriteString(key)
			result.WriteString("=")
			result.WriteString(structpath.EncodeMapKey(value))
			result.WriteString("]")
		} else if key, ok := node.StringKey(); ok {
			if i != 0 {
				result.WriteString(".")
			}
			result.WriteString(key)
		}
	}

	return result.String()
}

// sliceElement holds a slice element with its key information.
type sliceElement struct {
	keyField string
	keyValue string
	value    reflect.Value
}

// validateKeyFuncElementType verifies that the first element type in the sequence
// is assignable to the expected type. If the sequence is empty, it succeeds.
func validateKeyFuncElementType(seq reflect.Value, expected reflect.Type) error {
	if seq.Len() == 0 {
		return nil
	}
	elem := seq.Index(0)
	if !elem.Type().AssignableTo(expected) {
		return fmt.Errorf("KeyFunc expects %v, got %v", expected, elem.Type())
	}
	return nil
}

// diffSliceByKey compares two slices using the provided key function.
// Elements are matched by their (keyField, keyValue) pairs instead of by index.
// Duplicate keys are allowed and matched in order.
func diffSliceByKey(ctx *diffContext, path *structpath.PathNode, v1, v2 reflect.Value, keyFunc KeyFunc, changes *[]Change) error {
	caller, err := newKeyFuncCaller(keyFunc)
	if err != nil {
		return err
	}

	// Validate element types up-front to avoid runtime panics and to return a clear error.
	if err := validateKeyFuncElementType(v1, caller.argType); err != nil {
		return err
	}
	if err := validateKeyFuncElementType(v2, caller.argType); err != nil {
		return err
	}

	// Build lists of elements grouped by key, preserving order within each key
	elements1 := make(map[string][]sliceElement)
	elements2 := make(map[string][]sliceElement)
	seen := make(map[string]bool)
	var orderedKeys []string

	// Build from first slice
	for i := range v1.Len() {
		elem := v1.Index(i)
		keyField, keyValue := caller.call(elem.Interface())
		elements1[keyValue] = append(elements1[keyValue], sliceElement{keyField: keyField, keyValue: keyValue, value: elem})
		if !seen[keyValue] {
			seen[keyValue] = true
			orderedKeys = append(orderedKeys, keyValue)
		}
	}

	// Build from second slice
	for i := range v2.Len() {
		elem := v2.Index(i)
		keyField, keyValue := caller.call(elem.Interface())
		elements2[keyValue] = append(elements2[keyValue], sliceElement{keyField: keyField, keyValue: keyValue, value: elem})
		if !seen[keyValue] {
			seen[keyValue] = true
			orderedKeys = append(orderedKeys, keyValue)
		}
	}

	// Compare elements by key in original order
	for _, keyValue := range orderedKeys {
		list1 := elements1[keyValue]
		list2 := elements2[keyValue]

		var keyField string
		if len(list1) > 0 {
			keyField = list1[0].keyField
		} else {
			keyField = list2[0].keyField
		}

		// Match elements in order
		minLen := min(len(list1), len(list2))
		for i := range minLen {
			node := structpath.NewKeyValue(path, keyField, keyValue)
			if err := diffValues(ctx, node, list1[i].value, list2[i].value, changes); err != nil {
				return err
			}
		}

		// Handle extra elements in old (deleted)
		for i := minLen; i < len(list1); i++ {
			node := structpath.NewKeyValue(path, keyField, keyValue)
			*changes = append(*changes, Change{Path: node, Old: list1[i].value.Interface(), New: nil})
		}

		// Handle extra elements in new (added)
		for i := minLen; i < len(list2); i++ {
			node := structpath.NewKeyValue(path, keyField, keyValue)
			*changes = append(*changes, Change{Path: node, Old: nil, New: list2[i].value.Interface()})
		}
	}
	return nil
}
