package config

import (
	"iter"
	"reflect"
	"slices"
	"sync"

	"github.com/databricks/cli/libs/structs/structtag"
)

// ResourcesTypes returns a map from the configuration key of each Databricks resource
// group (for example "jobs" or "pipelines") to the Go type that represents a single
// resource instance inside that group (for example `resources.Job`).
var ResourcesTypes = sync.OnceValue(func() map[string]reflect.Type {
	res := make(map[string]reflect.Type, len(AllResourceTypes()))

	for _, rt := range AllResourceTypes() {
		res[rt.Key] = rt.Type

		// Automatically detect and add permissions field types
		// Look for a "Permissions" field in the resource type
		for _, resourceField := range reflect.VisibleFields(rt.Type) {
			if resourceField.Name == "Permissions" {
				permissionsKey := rt.Key + ".permissions"
				res[permissionsKey] = resourceField.Type
				continue
			}
			if resourceField.Name == "Grants" {
				grantsKey := rt.Key + ".grants"
				res[grantsKey] = resourceField.Type
			}
		}
	}

	return res
})

// ResourceType describes a resource type that can be declared under `resources:`.
// It corresponds to one field of [Resources].
type ResourceType struct {
	// Key is the key under `resources:` in the bundle configuration, for example "jobs".
	Key string

	// Type is the Go type of a single resource, for example `resources.Job`.
	Type reflect.Type

	// Zero is a pointer to a zero value of Type. It is shared by all callers:
	// use it to ask the type what it declares, never to hold or modify state.
	Zero ConfigResource

	// fieldIndex is the index of the corresponding field in [Resources].
	fieldIndex int
}

// AllResourceTypes returns every resource type in the field order of [Resources],
// which is the canonical order of resource types. The returned slice is shared
// and must not be modified.
var AllResourceTypes = sync.OnceValue(func() []ResourceType {
	rt := reflect.TypeFor[Resources]()
	res := make([]ResourceType, 0, rt.NumField())
	for i := range rt.NumField() {
		field := rt.Field(i)
		// Every field is a map[string]*T where *T implements ConfigResource;
		// TestResourcesFieldsAreConfigResourceMaps asserts this.
		elemType := field.Type.Elem().Elem()
		res = append(res, ResourceType{
			Key:        structtag.JSONTag(field.Tag.Get("json")).Name(),
			Type:       elemType,
			Zero:       reflect.New(elemType).Interface().(ConfigResource),
			fieldIndex: i,
		})
	}
	return res
})

// ResourceTypesImplementing yields, in canonical order, the key and zero value
// of every resource type whose zero value implements I. I is typically an
// optional interface through which a type declares a fact about itself.
func ResourceTypesImplementing[I any]() iter.Seq2[string, I] {
	return func(yield func(string, I) bool) {
		for _, rt := range AllResourceTypes() {
			v, ok := rt.Zero.(I)
			if ok && !yield(rt.Key, v) {
				return
			}
		}
	}
}

// ResourceTypeAs returns the zero value of the resource type with the given key as I.
// It returns false if key is not a resource type or if the type does not implement I.
func ResourceTypeAs[I any](key string) (I, bool) {
	types := AllResourceTypes()
	i := slices.IndexFunc(types, func(rt ResourceType) bool { return rt.Key == key })
	if i < 0 {
		var zero I
		return zero, false
	}
	v, ok := types[i].Zero.(I)
	return v, ok
}

// AssertResourceTypesImplement reports an error through t for every resource type
// that does not implement I and is not listed in except. It also reports every
// listed key that is not a resource type or whose type does implement I, so that
// opt-outs cannot go stale. It is meant for tests; t is typically a *testing.T,
// taken as an interface so that this package does not depend on "testing".
func AssertResourceTypesImplement[I any](t interface {
	Helper()
	Errorf(format string, args ...any)
}, except ...string,
) {
	t.Helper()
	name := reflect.TypeFor[I]().String()
	for _, key := range except {
		if _, ok := ResourceTypeAs[ConfigResource](key); !ok {
			t.Errorf("%q is listed as not implementing %s but is not a resource type", key, name)
		}
	}
	for _, rt := range AllResourceTypes() {
		_, implements := rt.Zero.(I)
		listed := slices.Contains(except, rt.Key)
		if !implements && !listed {
			t.Errorf("resource type %q (%s) does not implement %s", rt.Key, rt.Type, name)
		}
		if implements && listed {
			t.Errorf("resource type %q (%s) implements %s but is listed as not implementing it", rt.Key, rt.Type, name)
		}
	}
}
