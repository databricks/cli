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
	// index[i] is the Go field-index path of jsonNames[i], for reflect.Value.FieldByIndex.
	index [][]int
}

// registrations is populated only from init() (before any goroutine runs), then read
// concurrently, so it needs no lock.
var (
	registrations = map[reflect.Type]*keyedType{}
	keyFieldNames = map[string]struct{}{}
)

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
		keyFieldNames[name] = struct{}{}
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

// IsKeyField reports whether name is a key field of some registered type. Consumers
// that work on dynamic values without a Go type (e.g. configsync) use it to recognise
// an element's key field among its members.
func IsKeyField(name string) bool {
	_, ok := keyFieldNames[name]
	return ok
}

// ElementKey returns the identity of a keyed-slice element: the value of its first
// non-empty key field. ok is false if the element's type is not registered.
func ElementKey(elem reflect.Value) (value string, ok bool) {
	elem = deref(elem)
	kt := lookup(elem.Type())
	if kt == nil {
		return "", false
	}
	for _, index := range kt.index {
		if f := elem.FieldByIndex(index); f.String() != "" {
			return f.String(), true
		}
	}
	return "", true
}

// ElementHasValue reports whether a keyed-slice element carries value under any of
// its key fields. ok is false if the element's type is not registered.
func ElementHasValue(elem reflect.Value, value string) (matches, ok bool) {
	elem = deref(elem)
	kt := lookup(elem.Type())
	if kt == nil {
		return false, false
	}
	for _, index := range kt.index {
		if elem.FieldByIndex(index).String() == value {
			return true, true
		}
	}
	return false, true
}

func lookup(t reflect.Type) *keyedType {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return registrations[t]
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

// stringFieldIndex returns the field-index path of the string field with the given
// JSON name. It descends into flattened embeds (anonymous struct fields with no explicit
// JSON name, whose exported fields encoding/json promotes) — including embeds of
// unexported types, whose exported fields are still promoted. ok is false if no such
// field exists.
func stringFieldIndex(t reflect.Type, jsonName string) ([]int, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	for i := range t.NumField() {
		sf := t.Field(i)
		tagName := structtag.JSONTag(sf.Tag.Get("json")).Name()

		if sf.Anonymous && tagName == "" {
			et := sf.Type
			for et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				if index, ok := stringFieldIndex(et, jsonName); ok {
					return append([]int{i}, index...), true
				}
			}
			continue
		}

		if sf.IsExported() && sf.Type.Kind() == reflect.String && tagName == jsonName {
			return []int{i}, true
		}
	}
	return nil, false
}
