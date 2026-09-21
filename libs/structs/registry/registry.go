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
// This package only records the key field names; reading a field's value from an
// element is left to structaccess, so field resolution matches encoding/json exactly
// and stays consistent with the rest of the codebase.
//
// Types register their key fields from init(), keeping the declaration next to the
// type it describes:
//
//	func init() { registry.Register[jobs.Task]("task_key") }
package registry

import "reflect"

// registrations is populated only from init() (before any goroutine runs), then read
// concurrently, so it needs no lock.
var registrations = map[reflect.Type][]string{}

// Register declares the key fields (JSON names) that identify elements of slice
// element type T, in priority order. Intended to be called from init(). Pointer types
// are normalized to their element type, so Register[*T] and Register[T] agree.
func Register[T any](keyFields ...string) {
	registrations[deref(reflect.TypeFor[T]())] = keyFields
}

// KeyFields returns the key field JSON names for element type t (pointers
// dereferenced), or nil if t is not a registered keyed-slice element.
func KeyFields(t reflect.Type) []string {
	return registrations[deref(t)]
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
