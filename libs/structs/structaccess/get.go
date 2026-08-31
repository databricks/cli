package structaccess

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// NotFoundError is returned when a map key, slice index, or key-value selector is not found.
type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string {
	return e.msg
}

// GetByString returns the value at the given path inside v.
// This is a convenience function that parses the path string and calls Get.
func GetByString(v any, path string) (any, error) {
	if path == "" {
		return v, nil
	}

	pathNode, err := structpath.ParsePath(path)
	if err != nil {
		return nil, err
	}

	return Get(v, pathNode)
}

// getValue returns the reflect.Value at the given path inside v.
// This is the internal function that Get() wraps.
func getValue(v any, path *structpath.PathNode) (reflect.Value, error) {
	if path.IsRoot() {
		return reflect.ValueOf(v), nil
	}

	// Convert path to slice for easier iteration
	pathSegments := path.AsSlice()

	cur := reflect.ValueOf(v)
	for _, node := range pathSegments {
		// Note: wildcards cannot appear in PathNode (Parse rejects them)

		var ok bool
		cur, ok = deref(cur)
		if !ok {
			// cannot proceed further due to nil encountered at current location
			// There could be 2 cases: the type is correct but value is not found due to nil, in this case NotFoundError is 100% correct.
			// It could also be that path up to nil is correct, but not after. We don't know because we stop there. In this case NotFoundError refers to path up to nil.
			return reflect.Value{}, &NotFoundError{node.Parent().String() + ": cannot access nil value"}
		}

		if idx, isIndex := node.Index(); isIndex {
			// If cur is a struct with an EmbeddedSlice field, navigate through it.
			if cur.Kind() == reflect.Struct {
				if embed := findEmbedField(cur); embed.IsValid() {
					cur = embed
				}
			}
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				// Terraform represents single-block fields as lists and uses [0] to access them.
				// Treat [0] on a struct as a no-op so TF-style paths work against DABs structs.
				if idx == 0 && kind == reflect.Struct {
					continue
				}
				return reflect.Value{}, fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			if idx < 0 || idx >= cur.Len() {
				return reflect.Value{}, &NotFoundError{fmt.Sprintf("%s: index out of range, length is %d", node.String(), cur.Len())}
			}
			cur = cur.Index(idx)
			continue
		}

		if key, value, ok := node.KeyValue(); ok {
			// If cur is a struct with an EmbeddedSlice field, navigate through it.
			if cur.Kind() == reflect.Struct {
				if embed := findEmbedField(cur); embed.IsValid() {
					cur = embed
				}
			}
			nv, err := accessKeyValue(cur, key, value, node)
			if err != nil {
				return reflect.Value{}, err
			}
			cur = nv
			continue
		}

		key, ok := node.StringKey()
		if !ok {
			return reflect.Value{}, errors.New("unsupported path node type")
		}

		nv, err := accessKey(cur, key, node)
		if err != nil {
			return reflect.Value{}, err
		}
		cur = nv
	}

	return cur, nil
}

// Get returns the value at the given path inside v.
// Wildcards ("*" or "[*]") are not supported and return an error.
// Returns NotFoundError when a map key, slice index, or key-value selector is not found.
func Get(v any, path *structpath.PathNode) (any, error) {
	cur, err := getValue(v, path)
	if err != nil {
		return nil, err
	}

	// If the current value is invalid (e.g., omitted due to omitempty), return nil.
	if !cur.IsValid() {
		return nil, nil
	}

	// If the final value is a nil pointer or nil interface, return nil.
	if (cur.Kind() == reflect.Pointer || cur.Kind() == reflect.Interface) && cur.IsNil() {
		return nil, nil
	}

	// Return the resulting value as interface{}; do not force dereference of scalars.
	return cur.Interface(), nil
}

// accessKey returns the field or map entry value selected by key from v.
// v must be non-pointer, non-interface reflect.Value.
func accessKey(v reflect.Value, key string, path *structpath.PathNode) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Struct:
		fv, sf, owner, ok := findStructFieldByKey(v, key)
		if !ok {
			return reflect.Value{}, fmt.Errorf("%s: field %q not found in %s", path.String(), key, v.Type())
		}

		// ForceSendFields is only managed by the struct that declares the field.
		force := forceSendFieldsContains(owner, sf.Name)

		// Honor omitempty: if present and value is empty and not forced, treat as omitted (nil).
		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))
		if jsonTag.OmitEmpty() && !force {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return reflect.Value{}, nil
				}
				// Non-nil pointer: return the dereferenced value.
				// JSON omitempty only omits nil pointers, not pointers to zero values.
				// Returning the dereferenced value is consistent with GetStructDiff,
				// which recursively dereferences non-nil pointers.
				return fv.Elem(), nil
			} else if isEmptyForOmitEmpty(fv) {
				return reflect.Value{}, nil
			}
		}
		return fv, nil
	case reflect.Map:
		kt := v.Type().Key()
		if kt.Kind() != reflect.String {
			return reflect.Value{}, fmt.Errorf("%s: map key must be string, got %s", path.String(), kt)
		}
		mk := reflect.ValueOf(key)
		if kt != mk.Type() {
			mk = mk.Convert(kt)
		}
		mv := v.MapIndex(mk)
		if !mv.IsValid() {
			return reflect.Value{}, &NotFoundError{fmt.Sprintf("%s: key %q not found in map", path.String(), key)}
		}
		return mv, nil
	default:
		return reflect.Value{}, fmt.Errorf("%s: cannot access key %q on %s", path.String(), key, v.Kind())
	}
}

// accessKeyValue searches for an element in a slice/array where a field matching key has the given value.
// v must be a slice or array. Returns the first matching element.
func accessKeyValue(v reflect.Value, key, value string, path *structpath.PathNode) (reflect.Value, error) {
	kind := v.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return reflect.Value{}, fmt.Errorf("%s: cannot use key-value syntax on %s", path.String(), kind)
	}

	for i := range v.Len() {
		elem := v.Index(i)

		// Dereference pointers/interfaces in the element
		elemDeref, ok := deref(elem)
		if !ok {
			continue // Skip nil elements
		}

		// Element must be a struct to have fields
		if elemDeref.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("%s: key-value syntax requires slice elements to be structs, got %s", path.String(), elemDeref.Kind())
		}

		// Try to get the field value
		fieldVal, err := accessKey(elemDeref, key, path)
		if err != nil {
			continue // Field not found in this element, try next
		}

		// Check if the field value matches
		if !fieldVal.IsValid() {
			continue
		}

		// Only string fields are supported for key-value matching
		if fieldVal.Kind() != reflect.String {
			continue
		}

		if fieldVal.String() == value {
			return elem, nil
		}
	}

	return reflect.Value{}, &NotFoundError{fmt.Sprintf("%s: no element found with %s=%q", path.String(), key, value)}
}

// findFieldInStruct searches for a field by JSON key in a single struct (no embedding).
// Returns: fieldValue, structField, found
func findFieldInStruct(v reflect.Value, key string) (reflect.Value, reflect.StructField, bool) {
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		if sf.Anonymous { // skip embedded fields
			continue
		}

		// Read JSON tag using structtag helper
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" {
			name = ""
		}

		if sf.Name == EmbeddedSliceFieldName {
			continue // EmbeddedSlice fields are not accessible by name
		}
		if name != "" && name == key {
			// Skip fields marked as internal or readonly via bundle tag
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				continue
			}
			return v.Field(i), sf, true
		}
	}
	return reflect.Value{}, reflect.StructField{}, false
}

// findStructFieldByKey searches exported fields of struct v for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs recursively (flattening semantics), which
// FindStructFieldByKeyType does too: a bundle resource embeds a config struct that embeds
// the SDK request struct, so its fields sit two levels down.
// Returns: fieldValue, structField, owner (the struct value declaring the field), found
func findStructFieldByKey(v reflect.Value, key string) (reflect.Value, reflect.StructField, reflect.Value, bool) {
	// First pass: direct fields
	if fv, sf, found := findFieldInStruct(v, key); found {
		return fv, sf, v, true
	}

	// Second pass: search embedded anonymous structs (flattening semantics) breadth-first, one
	// level of embedding at a time. Not depth-first: encoding/json resolves a name declared at
	// two embedding depths in favour of the shallower one, so descending fully into the first
	// embed could pick a field three levels down over the same name two levels down in a
	// later one -- and then reading or writing the field would not be the field serialized
	// under that name.
	// Guards against a cyclic embedding (a struct embedding a pointer to itself), which would
	// otherwise enqueue the same type forever when the key is not found at all. Only types from
	// *earlier* levels are excluded: a type reachable twice within one level is a diamond, and
	// the two matches it produces are exactly the ambiguity json resolves by omitting the field.
	seen := map[reflect.Type]bool{v.Type(): true}
	level := embeddedStructs(v)
	for len(level) > 0 {
		var next []reflect.Value
		var found []struct {
			value reflect.Value
			field reflect.StructField
			owner reflect.Value
		}
		for _, fv := range level {
			if out, sf, ok := findFieldInStruct(fv, key); ok {
				found = append(found, struct {
					value reflect.Value
					field reflect.StructField
					owner reflect.Value
				}{out, sf, fv})
				continue
			}
			for _, deeper := range embeddedStructs(fv) {
				if seen[deeper.Type()] {
					continue
				}
				next = append(next, deeper)
			}
		}
		for _, fv := range next {
			seen[fv.Type()] = true
		}
		if len(found) == 1 {
			return found[0].value, found[0].field, found[0].owner, true
		}
		if len(found) > 1 {
			// Two embedded structs declare the same name at the same depth. encoding/json calls
			// that ambiguous and omits the field entirely, so there is no field to read or
			// write: picking one would target data that is never serialized.
			return reflect.Value{}, reflect.StructField{}, reflect.Value{}, false
		}
		level = next
	}

	return reflect.Value{}, reflect.StructField{}, reflect.Value{}, false
}

// embeddedStructs returns the anonymous struct fields of v, dereferenced, skipping any that
// cannot be descended into.
func embeddedStructs(v reflect.Value) []reflect.Value {
	var out []reflect.Value
	t := v.Type()
	for i := range t.NumField() {
		if !t.Field(i).Anonymous {
			continue
		}
		fv := v.Field(i)
		for fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				// Not initialized; can't descend.
				break
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		out = append(out, fv)
	}
	return out
}

// forceSendFields returns the ForceSendFields slice a struct declares itself. A struct that
// embeds another shadows it deliberately -- see resources.PostgresProjectConfig -- so only
// the declaring struct tracks a field of its own.
func forceSendFields(owner reflect.Value) reflect.Value {
	if !owner.IsValid() || owner.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	for i := range owner.Type().NumField() {
		field := owner.Type().Field(i)
		if field.Name == "ForceSendFields" && !field.Anonymous {
			return owner.Field(i)
		}
	}
	return reflect.Value{}
}

// forceSendFieldsContains reports whether a struct forces the named field to be sent.
func forceSendFieldsContains(owner reflect.Value, name string) bool {
	fsf := forceSendFields(owner)
	if !fsf.IsValid() {
		return false
	}
	fields, ok := reflect.TypeAssert[[]string](fsf)
	return ok && containsString(fields, name)
}

// containsString checks if a slice contains a specific string
func containsString(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

// isEmptyForOmitEmpty returns true if the value should be omitted by JSON omitempty.
// This matches JSON encoder behavior, which is different from reflect.IsZero() for slices/maps.
func isEmptyForOmitEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Struct:
		// Pointers to structs are not considered empty if pointer != nil
		// Structs as values are never empty and omitempty on them has no effect.
		return false
	default:
		return v.IsZero()
	}
}

// deref dereferences pointers and interfaces until it reaches a non-pointer, non-interface value.
// Returns ok=false if it encounters a nil pointer/interface.
func deref(v reflect.Value) (reflect.Value, bool) {
	for {
		switch v.Kind() {
		case reflect.Invalid:
			return v, false
		case reflect.Pointer:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		case reflect.Interface:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		default:
			return v, true
		}
	}
}
