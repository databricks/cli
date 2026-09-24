package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

func TestResourceFieldNameUsesExactJSONSkip(t *testing.T) {
	typ := reflect.TypeFor[struct {
		Skipped string `json:"-"`
		//nolint:govet,staticcheck // fixture intentionally exercises optioned dash tags
		Dash string `json:"-,omitempty"`
		//nolint:govet,staticcheck // fixture intentionally exercises optioned dash tags
		DashNoOpt string `json:"-,"`
		Normal    string `json:"normal,omitempty"`
	}]()

	name, ok := resourceFieldName(typ.Field(0))
	assert.False(t, ok)
	assert.Empty(t, name)
	name, ok = resourceFieldName(typ.Field(1))
	assert.True(t, ok)
	assert.Equal(t, "-", name)
	name, ok = resourceFieldName(typ.Field(2))
	assert.True(t, ok)
	assert.Equal(t, "-", name)
	name, ok = resourceFieldName(typ.Field(3))
	assert.True(t, ok)
	assert.Equal(t, "normal", name)
}

func TestResourcesTypesMap(t *testing.T) {
	assert.Greater(t, len(ResourcesTypes), 10, "expected ResourcesTypes to have more than 10 entries")

	typ, ok := ResourcesTypes["jobs"]
	assert.True(t, ok, "resources type for 'jobs' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[resources.Job](), typ, "resources type for 'jobs' mismatch")

	typ, ok = ResourcesTypes["jobs.permissions"]
	assert.True(t, ok, "resources type for 'jobs.permissions' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[[]resources.JobPermission](), typ, "resources type for 'jobs.permissions' mismatch")
}

func TestTopLevelResourcesHaveDirectIDField(t *testing.T) {
	for name, typ := range ResourcesTypes {
		if strings.Contains(name, ".") || name == "internal_immutable_snapshots" {
			continue
		}
		field, ok := typ.FieldByName("ID")
		if !assert.True(t, ok && len(field.Index) == 1, "%s must have a direct exported ID field", name) {
			continue
		}
		assert.Equal(t, `json:"id,omitempty" bundle:"readonly"`, string(field.Tag), "%s ID field has incorrect tags", name)
	}
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

// TestNoSameDepthJSONShadows verifies that encoding/json cannot encounter an
// ambiguous field at any containing struct, not only at the resource root.
func TestNoSameDepthJSONShadows(t *testing.T) {
	rt := reflect.TypeFor[Resources]()
	var collisions []string

	for f := range rt.Fields() {
		et := f.Type.Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(f.Tag.Get("json")).Name()
		directNames := map[string]bool{}
		for field := range et.Fields() {
			if field.PkgPath != "" || structaccess.IsSkippedField(field) {
				continue
			}
			name := structtag.JSONTag(field.Tag.Get("json")).Name()
			if name == "" {
				name = field.Name
			}
			directNames[name] = true
		}
		visits := map[string]int{}
		_ = structwalk.WalkType(et, func(path *structpath.PatternNode, _ reflect.Type, sf *reflect.StructField) bool {
			if sf != nil && path != nil {
				visits[path.String()]++
			}
			return true
		})
		for name, count := range visits {
			// A direct field at the resource root intentionally shadows promoted
			// fields. At every nested path there is no such shadowing rule.
			if count > 1 && strings.Contains(name, ".") && !directNames[strings.SplitN(name, ".", 2)[0]] {
				collisions = append(collisions, fmt.Sprintf("%s.%s visited %d times", group, name, count))
			}
		}
	}

	slices.Sort(collisions)
	assert.Empty(t, collisions, "same-depth JSON name collisions found")
}
