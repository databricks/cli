package resources_test

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bindFixture is a resource that TestResourcesBindSupport binds to the ID id.
// expect mocks the API calls that the resource's Exists makes so that it finds
// the resource.
type bindFixture struct {
	resource config.ConfigResource
	id       string
	expect   func(m *mocks.MockWorkspaceClient)
}

// notBindable declares that the type of resource cannot be bound, and why.
type notBindable struct {
	resource config.ConfigResource
	reason   string
}

var (
	bindFixtures []bindFixture
	notBindables []notBindable
)

// registerBindFixture is called from the init function of a resource type's
// <key>_bind_test.go. Every resource type registers either a bind fixture or
// a reason with registerNotBindable.
func registerBindFixture(f bindFixture) {
	bindFixtures = append(bindFixtures, f)
}

func registerNotBindable(resource config.ConfigResource, reason string) {
	notBindables = append(notBindables, notBindable{resource: resource, reason: reason})
}

func TestResourcesBindSupport(t *testing.T) {
	// A registration is matched to its resource type by Go type, so that a
	// registration for something that is not a resource type fails here.
	keyByType := map[reflect.Type]string{}
	for _, rt := range config.AllResourceTypes() {
		keyByType[rt.Type] = rt.Key
	}
	keyOf := func(resource config.ConfigResource) string {
		typ := reflect.TypeOf(resource)
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		key, ok := keyByType[typ]
		require.True(t, ok, "%T is not the type of a field of config.Resources", resource)
		return key
	}

	fixtures := map[string]bindFixture{}
	for _, f := range bindFixtures {
		key := keyOf(f.resource)
		_, ok := fixtures[key]
		require.False(t, ok, "resource type %q registers more than one bind fixture", key)
		fixtures[key] = f
	}

	unbindable := map[string]bool{}
	for _, n := range notBindables {
		key := keyOf(n.resource)
		require.NotEmpty(t, n.reason, "resource type %q is declared not bindable without a reason", key)
		require.False(t, unbindable[key], "resource type %q is declared not bindable more than once", key)
		unbindable[key] = true
	}

	for _, rt := range config.AllResourceTypes() {
		_, hasFixture := fixtures[rt.Key]
		if hasFixture == unbindable[rt.Key] {
			t.Errorf("resource type %q must register exactly one of a bind fixture (registerBindFixture) or a reason why it cannot be bound (registerNotBindable), in %s_bind_test.go", rt.Key, rt.Key)
		}
	}

	// Bind looks the resource up among all resources of the bundle, so every
	// fixture goes into the same Resources.
	r := &config.Resources{}
	rv := reflect.ValueOf(r).Elem()
	fieldByType := map[reflect.Type]reflect.Value{}
	for _, field := range rv.Fields() {
		fieldByType[field.Type().Elem()] = field
	}
	for _, f := range fixtures {
		field, ok := fieldByType[reflect.TypeOf(f.resource)]
		require.True(t, ok, "%T is not the type of a field of config.Resources", f.resource)
		group := reflect.MakeMap(field.Type())
		group.SetMapIndex(reflect.ValueOf("my_"+f.resource.ResourceDescription().SingularName), reflect.ValueOf(f.resource))
		field.Set(group)
	}

	for _, rt := range config.AllResourceTypes() {
		f, ok := fixtures[rt.Key]
		if !ok {
			continue
		}
		t.Run(rt.Key, func(t *testing.T) {
			m := mocks.NewMockWorkspaceClient(t)
			f.expect(m)

			// bind operation requires resource to be returned from ResolveKey
			res, _, err := r.ResolveKey("my_" + rt.Description.SingularName)
			assert.NoError(t, err)

			// bind operation requires Exists to return true
			exists, err := res.Exists(t.Context(), m.WorkspaceClient, f.id)
			assert.NoError(t, err)
			assert.True(t, exists)
		})
	}
}
