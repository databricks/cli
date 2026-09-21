// Package registry records, per Go type, which fields identify the elements of a
// keyed slice.
//
// Diffing and path resolution match keyed-slice elements by the value of these
// fields instead of by position, and address an element in a path as [='value']:
// the key field is omitted from the path because resolution is by value, using this
// registry. Several key fields express a one-of identity — a permission is a
// user_name OR a service_principal_name OR a group_name — and an element's identity
// is the value of its first non-empty key field.
//
// The design assumes an element's key value is unique across its key fields within a
// slice: a user, service principal, and group never share the same value string
// (emails, application IDs, and group names occupy disjoint value spaces), so a
// value alone unambiguously identifies the element.
//
// Types register their key fields from init(), keeping the declaration next to the
// type it describes:
//
//	func init() { registry.Register[jobs.Task]("task_key") }
package registry

import (
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/structs/structtag"
)

// keyedType is the resolved key definition for one element type.
type keyedType struct {
	// jsonNames are the key field JSON names, in declaration (priority) order.
	jsonNames []string
	// index[i] is the Go field-index path of jsonNames[i], for fieldByIndex.
	index [][]int
}

// registrations is populated only from init() (before any goroutine runs), then read
// concurrently, so it needs no lock.
var registrations = map[reflect.Type]*keyedType{}

// Register declares the key fields (JSON names) that identify elements of slice
// element type T. Intended to be called from init(). It panics if a name does not
// resolve to a string field of T, surfacing typos at startup rather than as silent
// mismatches at diff time.
func Register[T any](keyFields ...string) {
	t := reflect.TypeFor[T]()
	kt := &keyedType{jsonNames: keyFields}
	for _, name := range keyFields {
		index, ok := stringFieldIndex(t, name)
		if !ok {
			panic("registry: " + t.String() + " has no string field with json name " + strconv.Quote(name))
		}
		kt.index = append(kt.index, index)
	}
	registrations[t] = kt
}

// KeyFields returns the key field JSON names for element type t (pointers
// dereferenced), or nil if t is not a registered keyed-slice element.
func KeyFields(t reflect.Type) []string {
	if kt := lookup(t); kt != nil {
		return kt.jsonNames
	}
	return nil
}

// ElementKey returns the identity of a keyed-slice element: the value of its first
// non-empty key field. ok is false if the element's type is not registered (or the
// element is a nil pointer).
func ElementKey(elem reflect.Value) (value string, ok bool) {
	elem, valid := deref(elem)
	if !valid {
		return "", false
	}
	kt := lookup(elem.Type())
	if kt == nil {
		return "", false
	}
	for _, index := range kt.index {
		f, ok := fieldByIndex(elem, index)
		if ok && f.String() != "" {
			return f.String(), true
		}
	}
	return "", true
}

func lookup(t reflect.Type) *keyedType {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return registrations[t]
}

// deref follows pointers and interfaces to the underlying value. valid is false if a
// nil is encountered along the way.
func deref(v reflect.Value) (reflect.Value, bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	return v, v.IsValid()
}

// fieldByIndex reads the field at the given index path, following pointer embeds.
// ok is false if a nil pointer is encountered along the path (rather than panicking
// like reflect.Value.FieldByIndex).
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 {
			var valid bool
			if v, valid = deref(v); !valid {
				return reflect.Value{}, false
			}
		}
		v = v.Field(x)
	}
	return v, v.Kind() == reflect.String
}

// stringFieldIndex returns the field-index path of the string field with the given
// JSON name. Direct fields are resolved first, then flattened embeds breadth-first
// (anonymous struct fields with no explicit JSON name, whose exported fields
// encoding/json promotes), so a field shadowed by an outer one of the same JSON name
// resolves to the outer field that JSON serializes. ok is false if no such field
// exists.
func stringFieldIndex(t reflect.Type, jsonName string) ([]int, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}

	// Direct fields first (shallower wins over promoted fields).
	for i := range t.NumField() {
		sf := t.Field(i)
		if isFlattenedEmbed(sf) {
			continue
		}
		if sf.IsExported() && sf.Type.Kind() == reflect.String && structtag.JSONTag(sf.Tag.Get("json")).Name() == jsonName {
			return []int{i}, true
		}
	}

	// Then promoted fields of flattened embeds, breadth-first.
	level := embedsOf(t, nil)
	for len(level) > 0 {
		var next []embed
		for _, e := range level {
			for i := range e.typ.NumField() {
				sf := e.typ.Field(i)
				if isFlattenedEmbed(sf) {
					continue
				}
				if sf.IsExported() && sf.Type.Kind() == reflect.String && structtag.JSONTag(sf.Tag.Get("json")).Name() == jsonName {
					return append(append([]int{}, e.prefix...), i), true
				}
			}
			next = append(next, embedsOf(e.typ, e.prefix)...)
		}
		level = next
	}
	return nil, false
}

type embed struct {
	prefix []int
	typ    reflect.Type
}

// embedsOf returns the flattened embeds of struct type t, each with its field-index
// prefix relative to the original type.
func embedsOf(t reflect.Type, prefix []int) []embed {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var result []embed
	for i := range t.NumField() {
		sf := t.Field(i)
		if !isFlattenedEmbed(sf) {
			continue
		}
		et := sf.Type
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() == reflect.Struct {
			result = append(result, embed{prefix: append(append([]int{}, prefix...), i), typ: et})
		}
	}
	return result
}

// isFlattenedEmbed reports whether sf is an anonymous struct field with no explicit
// JSON name, whose exported fields encoding/json promotes to the outer type.
func isFlattenedEmbed(sf reflect.StructField) bool {
	if !sf.Anonymous {
		return false
	}
	return structtag.JSONTag(sf.Tag.Get("json")).Name() == ""
}
