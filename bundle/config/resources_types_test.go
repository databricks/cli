package config

import (
	"encoding/json"
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
