package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/structs/structtag"
)

func TestResourcesTypesMap(t *testing.T) {
	assert.Greater(t, len(ResourcesTypes), 10, "expected ResourcesTypes to have more than 10 entries")

	typ, ok := ResourcesTypes["jobs"]
	assert.True(t, ok, "resources type for 'jobs' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[resources.Job](), typ, "resources type for 'jobs' mismatch")

	typ, ok = ResourcesTypes["jobs.permissions"]
	assert.True(t, ok, "resources type for 'jobs.permissions' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[[]resources.JobPermission](), typ, "resources type for 'jobs.permissions' mismatch")
}

// TestResourceTypesZeroValueFieldsSerialize guards against the ForceSendFields
// routing bug fixed in libs/dyn/convert: a field declared in a struct embedded
// more than one level deep (e.g. PostgresProject -> PostgresProjectConfig ->
// ProjectSpec) had its zero value recorded in the wrong struct's ForceSendFields,
// which the SDK marshaler rejects with "field X cannot be found in struct Y".
// The direct engine hits this path when it serializes planned state to JSON.
//
// For every registered resource type it sets every omitempty scalar field (at any
// depth) to its zero value, converts via ToTyped, and marshals - the same round
// trip the direct engine performs. Any newly added resource whose wrapper embeds
// an SDK spec is covered automatically.
func TestResourceTypesZeroValueFieldsSerialize(t *testing.T) {
	names := make([]string, 0, len(ResourcesTypes))
	for name := range ResourcesTypes {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			typ := ResourcesTypes[name]
			zeros := zeroValueScalars(typ, 0, map[reflect.Type]bool{})
			if zeros.Kind() != dyn.KindMap {
				return
			}

			ptr := reflect.New(typ)
			require.NoError(t, convert.ToTyped(ptr.Interface(), zeros))

			_, err := json.Marshal(ptr.Interface())
			require.NoError(t, err)
		})
	}
}

// zeroValueScalars builds a [dyn.Value] map that sets every omitempty scalar field
// reachable through embedded anonymous structs to its zero value. Those are exactly
// the fields the convert layer records in ForceSendFields, so they exercise the
// routing logic. depth and seen bound recursion against deep or recursive types.
func zeroValueScalars(t reflect.Type, depth int, seen map[reflect.Type]bool) dyn.Value {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || depth > 6 || seen[t] {
		return dyn.NilValue
	}
	seen[t] = true
	defer delete(seen, t)

	m := dyn.NewMapping()
	for f := range t.Fields() {
		if f.Anonymous {
			if sub := zeroValueScalars(f.Type, depth+1, seen); sub.Kind() == dyn.KindMap {
				for _, p := range sub.MustMap().Pairs() {
					m.SetLoc(p.Key.MustString(), nil, p.Value)
				}
			}
			continue
		}

		tag := structtag.JSONTag(f.Tag.Get("json"))
		name := tag.Name()
		if name == "" || name == "-" || !f.IsExported() || !tag.OmitEmpty() {
			continue
		}

		switch f.Type.Kind() {
		case reflect.Bool:
			m.SetLoc(name, nil, dyn.V(false))
		case reflect.String:
			m.SetLoc(name, nil, dyn.V(""))
		case reflect.Int, reflect.Int32, reflect.Int64:
			m.SetLoc(name, nil, dyn.V(int64(0)))
		case reflect.Float32, reflect.Float64:
			m.SetLoc(name, nil, dyn.V(float64(0)))
		default:
			// Only basic types are eligible for ForceSendFields; skip the rest.
		}
	}
	return dyn.V(m)
}

// knownSameDepthCollisions lists the existing same-depth json name collisions.
// These are bugs — encoding/json calls the name ambiguous and neither field is
// reachable — but fixing them requires a breaking change to the bundle YAML
// format, so they are listed here rather than silently tolerated. Every entry
// must describe where the collision comes from; fixing one removes its entry.
//
// Fix: add an explicit depth-0 field on the resource type that shadows both
// embedded declarations (e.g. `ID string \`json:"id,omitempty" bundle:"readonly"\“
// directly on the resource struct).
var knownSameDepthCollisions = map[string][]string{
	// BaseResource.ID and pipelines.CreatePipeline.Id both carry json:"id".
	"pipelines": {"id"},
	// BaseResource.ID and apps.App.Id carry json:"id";
	// BaseResource.URL and apps.App.Url carry json:"url".
	"apps": {"id", "url"},
	// BaseResource.ID and sql.AlertV2.Id carry json:"id".
	"alerts": {"id"},
}

func TestNoSameDepthJSONShadows(t *testing.T) {
	rt := reflect.TypeFor[Resources]()
	var collisions []string

	for i := range rt.NumField() {
		f := rt.Field(i)
		et := f.Type.Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(f.Tag.Get("json")).Name()
		for _, c := range sameDepthCollisions(et) {
			collisions = append(collisions,
				fmt.Sprintf("%s: json name %q declared by %s and %s at the same embedding depth",
					group, c.name, c.typeA, c.typeB))
		}
	}

	assert.Empty(t, collisions,
		"same-depth json name collisions found — encoding/json calls these ambiguous "+
			"and serializes neither; structaccess cannot read or write them either")
}

type collision struct {
	name, typeA, typeB string
}

// sameDepthCollisions returns json names declared at the same embedding depth
// by two or more anonymous embedded structs that are NOT already shadowed by a
// direct field on t itself. A same-depth collision is only a problem when there
// is no depth-0 field that resolves the ambiguity; if one exists (e.g. App.URL
// at depth 0 shadows both BaseResource.URL and apps.App.Url at depth 1),
// encoding/json and structaccess both use the depth-0 field correctly.
func sameDepthCollisions(t reflect.Type) []collision {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	// Depth-0 direct fields shadow any same-depth collision at deeper levels.
	depth0 := map[string]bool{}
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" || sf.Anonymous || sf.Name == "ForceSendFields" {
			continue
		}
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "" {
			name = sf.Name
		}
		if name != "-" {
			depth0[name] = true
		}
	}

	var result []collision
	seen := map[reflect.Type]bool{t: true}
	level := embeddedTypes(t)

	for len(level) > 0 {
		nameToTypes := map[string][]string{}
		for _, ft := range level {
			for sf := range ft.Fields() {
				if sf.PkgPath != "" || sf.Anonymous {
					continue
				}
				name := structtag.JSONTag(sf.Tag.Get("json")).Name()
				if name == "" {
					name = sf.Name
				}
				if name == "-" || sf.Name == "ForceSendFields" {
					continue
				}
				nameToTypes[name] = append(nameToTypes[name], ft.String())
			}
		}
		for name, types := range nameToTypes {
			if len(types) > 1 && !depth0[name] {
				result = append(result, collision{name: name, typeA: types[0], typeB: types[1]})
			}
		}

		var next []reflect.Type
		for _, ft := range level {
			for _, embedded := range embeddedTypes(ft) {
				if !seen[embedded] {
					seen[embedded] = true
					next = append(next, embedded)
				}
			}
		}
		level = next
	}
	return result
}

func embeddedTypes(t reflect.Type) []reflect.Type {
	var out []reflect.Type
	for sf := range t.Fields() {
		if !sf.Anonymous {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			out = append(out, ft)
		}
	}
	return out
}

// TestResourceIDFieldTags asserts that every resource type exposes the bundle
// tracking ID with exactly the right json and bundle tags. The field must be
// json:"id,omitempty" (so it round-trips through the bundle state file) and
// bundle:"readonly" (so users can reference ${resources.<key>.id} but cannot
// set it). It must be a direct depth-0 field, not promoted from BaseResource
// or an SDK embed, to avoid same-depth collisions.
func TestResourceIDFieldTags(t *testing.T) {
	rt := reflect.TypeFor[Resources]()
	for i := range rt.NumField() {
		f := rt.Field(i)
		et := f.Type.Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(f.Tag.Get("json")).Name()

		// Skip resource types that don't embed BaseResource (internal infra types
		// like Snapshot have no user-facing ID).
		hasBaseResource := false
		for j := range et.NumField() {
			if et.Field(j).Anonymous && et.Field(j).Type.Name() == "BaseResource" {
				hasBaseResource = true
				break
			}
		}
		if !hasBaseResource {
			continue
		}

		t.Run(group, func(t *testing.T) {
			// The ID field must be declared directly on the resource struct,
			// not promoted from an anonymous embed.
			sf, ok := et.FieldByName("ID")
			require.True(t, ok, "%s must have a direct ID field", group)
			assert.False(t, sf.Anonymous, "%s.ID must not be anonymous", group)
			assert.Empty(t, sf.Index[1:], "%s.ID must be at depth 0 (got index %v)", group, sf.Index)

			assert.Equal(t, "id,omitempty", sf.Tag.Get("json"),
				"%s.ID json tag must be \"id,omitempty\"", group)
			assert.Equal(t, "readonly", sf.Tag.Get("bundle"),
				"%s.ID bundle tag must be \"readonly\"", group)
		})
	}
}
