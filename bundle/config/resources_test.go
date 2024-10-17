package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test ensures that all resources have a custom marshaller and unmarshaller.
// This is required because DABs resources map to Databricks APIs, and they do so
// by embedding the corresponding Go SDK structs.
//
// Go SDK structs often implement custom marshalling and unmarshalling methods (based on the API specifics).
// If the Go SDK struct implements custom marshalling and unmarshalling and we do not
// for the resources at the top level, marshalling and unmarshalling operations will panic.
// Thus we will be overly cautious and ensure that all resources need a custom marshaller and unmarshaller.
//
// Why do we not assert this using an interface to assert MarshalJSON and UnmarshalJSON
// are implemented at the top level?
// If a method is implemented for an embedded struct, the top level struct will
// also have that method and satisfy the interface. This is why we cannot assert
// that the methods are implemented at the top level using an interface.
//
// Why don't we use reflection to assert that the methods are implemented at the
// top level?
// Same problem as above, the golang reflection package does not seem to provide
// a way to directly assert that MarshalJSON and UnmarshalJSON are implemented
// at the top level.
func TestCustomMarshallerIsImplemented(t *testing.T) {
	r := Resources{}
	rt := reflect.TypeOf(r)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// Fields in Resources are expected be of the form map[string]*resourceStruct
		assert.Equal(t, field.Type.Kind(), reflect.Map, "Resource %s is not a map", field.Name)
		kt := field.Type.Key()
		assert.Equal(t, kt.Kind(), reflect.String, "Resource %s is not a map with string keys", field.Name)
		vt := field.Type.Elem()
		assert.Equal(t, vt.Kind(), reflect.Ptr, "Resource %s is not a map with pointer values", field.Name)

		// Marshalling a resourceStruct will panic if resourceStruct does not have a custom marshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom marshaller.
		// Eg: resource.Job implements MarshalJSON
		v := reflect.Zero(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			json.Marshal(v)
		}, "Resource %s does not have a custom marshaller", field.Name)

		// Unmarshalling a *resourceStruct will panic if the resource does not have a custom unmarshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom unmarshaller.
		// Eg: *resource.Job implements UnmarshalJSON
		v = reflect.New(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			json.Unmarshal([]byte("{}"), v)
		}, "Resource %s does not have a custom unmarshaller", field.Name)
	}
}

func TestResourcesAllResourcesCompleteness(t *testing.T) {
	r := Resources{}
	rt := reflect.TypeOf(r)

	// Collect set of includes resource types
	var types []string
	for _, group := range r.AllResources() {
		types = append(types, group.Description.PluralName)
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		jsonTag := field.Tag.Get("json")

		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		assert.Contains(t, types, jsonTag, "Field %s is missing in AllResources", field.Name)
	}
}

func TestSupportedResources(t *testing.T) {
	// Please add your resource to the SupportedResources() function in resources.go if you add a new resource.
	actual := SupportedResources()

	typ := reflect.TypeOf(Resources{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTags := strings.Split(field.Tag.Get("json"), ",")
		pluralName := jsonTags[0]
		assert.Equal(t, actual[pluralName].PluralName, pluralName)
	}
}
