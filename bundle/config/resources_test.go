package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/require"

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
	rt := reflect.TypeFor[Resources]()

	for field := range rt.Fields() {
		// Fields in Resources are expected be of the form map[string]*resourceStruct
		assert.Equal(t, reflect.Map, field.Type.Kind(), "Resource %s is not a map", field.Name)
		kt := field.Type.Key()
		assert.Equal(t, reflect.String, kt.Kind(), "Resource %s is not a map with string keys", field.Name)
		vt := field.Type.Elem()
		assert.Equal(t, reflect.Pointer, vt.Kind(), "Resource %s is not a map with pointer values", field.Name)

		// Marshalling a resourceStruct will panic if resourceStruct does not have a custom marshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom marshaller.
		// Eg: resource.Job implements MarshalJSON
		v := reflect.Zero(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			_, err := json.Marshal(v)
			assert.NoError(t, err)
		}, "Resource %s does not have a custom marshaller", field.Name)

		// Unmarshalling a *resourceStruct will panic if the resource does not have a custom unmarshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom unmarshaller.
		// Eg: *resource.Job implements UnmarshalJSON
		v = reflect.New(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			err := json.Unmarshal([]byte("{}"), v)
			assert.NoError(t, err)
		}, "Resource %s does not have a custom unmarshaller", field.Name)
	}
}

func TestResourcesAllResources(t *testing.T) {
	r := Resources{
		// A nil entry comes from a null in Python-generated configuration.
		Jobs:      map[string]*resources.Job{"my_job": {}, "null_job": nil},
		Snapshots: map[string]*resources.Snapshot{"my_snapshot": {}},
	}

	groups := r.AllResources()
	require.Len(t, groups, len(AllResourceTypes()))

	nonEmpty := map[string]map[string]ConfigResource{}
	for i, rt := range AllResourceTypes() {
		assert.Equal(t, rt.Key, groups[i].Description.PluralName)
		if len(groups[i].Resources) > 0 {
			nonEmpty[rt.Key] = groups[i].Resources
		}
	}
	assert.Equal(t, map[string]map[string]ConfigResource{
		"jobs":                         {"my_job": r.Jobs["my_job"], "null_job": (*resources.Job)(nil)},
		"internal_immutable_snapshots": {"my_snapshot": r.Snapshots["my_snapshot"]},
	}, nonEmpty)

	// The nil entry is kept as a typed nil, which lifecycle.depends_on validation relies on.
	nullJob, ok := nonEmpty["jobs"]["null_job"]
	require.True(t, ok)
	require.IsType(t, (*resources.Job)(nil), nullJob)
	assert.True(t, reflect.ValueOf(nullJob).IsNil())
}

func TestSupportedResources(t *testing.T) {
	actual := SupportedResources()
	assert.Len(t, actual, len(AllResourceTypes()))
	for _, rt := range AllResourceTypes() {
		assert.Equal(t, rt.Description, actual[rt.Key])
	}
}

// Every resource type declares in its own file either the URL pattern of its
// page in the workspace UI or that it has no page.
func TestEveryResourceTypeDeclaresWorkspaceURL(t *testing.T) {
	AssertResourceTypesImplement[resources.WorkspaceURLDeclarer](t)
}

// A type that declares a page initializes the URL of its resources, and a type
// that declares none does not.
func TestResourceTypesWithWorkspacePageInitializeURL(t *testing.T) {
	for key, d := range ResourceTypesImplementing[resources.WorkspaceURLDeclarer]() {
		_, hasPage := d.WorkspaceURL()
		_, initializesURL := d.(resources.URLInitializer)
		assert.Equalf(t, hasPage, initializesURL, "resource type %q declares a workspace page: %v, implements resources.URLInitializer: %v", key, hasPage, initializesURL)
	}
}

func TestWorkspaceURLPatternsHaveOneID(t *testing.T) {
	patterns := WorkspaceURLPatterns()
	assert.NotEmpty(t, patterns)
	for key, p := range patterns {
		assert.Equalf(t, 1, strings.Count(p.Path, "%s"), "URL pattern %q of resource type %q must hold exactly one %%s", p.Path, key)
	}
}

func TestAllInteralResourcesAreMarkedAsInternal(t *testing.T) {
	internalResourceKeys := map[string]reflect.Type{}
	err := structwalk.WalkType(reflect.TypeFor[Resources](), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path.Len() > 2 {
			return false
		}
		if field == nil {
			return true
		}
		tag := field.Tag.Get("bundle")
		if structtag.BundleTag(tag).Internal() {
			internalResourceKeys[field.Name] = typ
		}
		return true
	})
	assert.NoError(t, err)

	for key, typ := range internalResourceKeys {
		r := reflect.MakeMap(typ)
		r.SetMapIndex(reflect.ValueOf("my_resources"), reflect.New(typ.Elem()).Elem())

		resources := &Resources{}
		res := reflect.ValueOf(resources).Elem()
		field := res.FieldByName(key)
		if !field.IsValid() && !field.CanSet() {
			t.Fatalf("Field %s is not valid", key)
		}
		field.Set(r)
		assert.True(t, resources.HasInternalResources())
	}
}

func TestResolveKey(t *testing.T) {
	r := &Resources{
		Jobs:      map[string]*resources.Job{"foo": {}, "thing": {}},
		Pipelines: map[string]*resources.Pipeline{"bar": {}, "thing": {}},
	}

	tests := []struct {
		ref      string
		wantType string
		wantKey  string
		wantErr  string
	}{
		{ref: "foo", wantType: "jobs", wantKey: "foo"},
		{ref: "jobs.foo", wantType: "jobs", wantKey: "foo"},
		{ref: "pipelines.thing", wantType: "pipelines", wantKey: "thing"},
		{ref: "missing", wantErr: "no such resource: missing"},
		{ref: "jobs.bar", wantErr: "no such resource: jobs.bar"},
		{ref: "nosuchtype.foo", wantErr: "no such resource: nosuchtype.foo"},
		{ref: "thing", wantErr: "ambiguous resource: thing (can resolve to jobs.thing, pipelines.thing); use a qualified name to disambiguate"},
	}

	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			res, key, err := r.ResolveKey(tc.ref)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, res.ResourceDescription().PluralName)
			assert.Equal(t, tc.wantKey, key)
		})
	}
}

func TestResolveDependsOn(t *testing.T) {
	r := &Resources{
		Jobs:      map[string]*resources.Job{"foo": {}, "thing": {}},
		Pipelines: map[string]*resources.Pipeline{"thing": {}},
		Schemas:   map[string]*resources.Schema{"bar": {}},
	}

	tests := []struct {
		ref     string
		want    string
		wantErr string
	}{
		{ref: "foo", want: "resources.jobs.foo"},
		{ref: "jobs.foo", want: "resources.jobs.foo"},
		{ref: "schemas.bar.grants", want: "resources.schemas.bar.grants"},
		{ref: "jobs.foo.permissions", want: "resources.jobs.foo.permissions"},
		{ref: "missing", wantErr: "no such resource: missing"},
		{ref: "thing", wantErr: "ambiguous resource: thing (can resolve to jobs.thing, pipelines.thing); use a qualified name to disambiguate"},
		{ref: "jobs.foo.grants", wantErr: "no such resource: jobs.foo.grants"},
		{ref: "schemas.missing.grants", wantErr: "no such resource: schemas.missing"},
		{ref: "bar.grants", wantErr: "no such resource: bar.grants"},
		{ref: "a.b.c", wantErr: "no such resource: a.b.c"},
		{ref: "a.b.c.d", wantErr: "no such resource: a.b.c.d"},
	}

	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			got, err := r.ResolveDependsOn(tc.ref)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
