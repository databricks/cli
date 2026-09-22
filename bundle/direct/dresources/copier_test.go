package dresources

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoRedundantRemapState enforces the "only override when needed" rule: a resource that
// keeps a hand-written RemapState must actually do something the auto-generated copier cannot.
// For every resource that still has a RemapState whose remote and state types differ, it fully
// populates a remote value and fails if the copier would produce an identical result — meaning
// the method is dead boilerplate that should be deleted (see buildCopiers).
func TestNoRedundantRemapState(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var redundant []string

	for resourceType, resource := range SupportedResources {
		implType := reflect.TypeOf(resource)
		if seen[implType] {
			continue
		}
		seen[implType] = true

		m := reflect.ValueOf(resource).MethodByName("RemapState")
		if !m.IsValid() {
			continue
		}
		remoteType := m.Type().In(0)
		stateType := m.Type().Out(0)
		if remoteType == stateType {
			continue // identity, the copier is not involved
		}

		copier, err := compileCopier(remoteType, stateType)
		if err != nil {
			continue // copier cannot handle it, so the override is required
		}

		remote := reflect.New(remoteType.Elem())
		fillValue(remote.Elem(), 0, map[reflect.Type]bool{})

		want := m.Call([]reflect.Value{remote})[0].Interface()
		got := copier.copy(remote.Interface())
		if reflect.DeepEqual(want, got) {
			redundant = append(redundant, resourceType)
		}
	}

	sort.Strings(redundant)
	assert.Empty(t, redundant, "these resources have a RemapState the auto-copier reproduces exactly; delete the method and let buildCopiers handle it")
}

type copierKindA string

type copierKindB string

type copierRemote struct {
	Name            string      `json:"name"`
	Extra           string      `json:"extra"` // absent from state -> dropped
	Kind            copierKindA `json:"kind"`  // same underlying type as state, distinct name -> converted
	ForceSendFields []string
}

type copierState struct {
	Name            string      `json:"name"`
	Missing         string      `json:"missing"` // absent from remote -> left zero
	Kind            copierKindB `json:"kind"`
	ForceSendFields []string
}

func TestRemapCopierCopy(t *testing.T) {
	copier, err := compileCopier(reflect.TypeFor[*copierRemote](), reflect.TypeFor[*copierState]())
	require.NoError(t, err)

	remote := &copierRemote{
		Name:            "n",
		Extra:           "e",
		Kind:            "classic",
		ForceSendFields: []string{"Name", "Extra", "Kind"},
	}
	got := copier.copy(remote).(*copierState)

	assert.Equal(t, "n", got.Name)
	assert.Empty(t, got.Missing)                                   // absent from remote
	assert.Equal(t, copierKindB("classic"), got.Kind)              // converted across enum types
	assert.Equal(t, []string{"Name", "Kind"}, got.ForceSendFields) // "Extra" filtered out (not a state field)
}

type copierUnsafeRemote struct {
	N int `json:"n"`
}

type copierUnsafeState struct {
	N string `json:"n"`
}

func TestRemapCopierCompileRejectsUnsafe(t *testing.T) {
	_, err := compileCopier(reflect.TypeFor[*copierUnsafeRemote](), reflect.TypeFor[*copierUnsafeState]())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "implement RemapState")
}

func TestSafeConvert(t *testing.T) {
	tests := []struct {
		name string
		dst  reflect.Type
		src  reflect.Type
		want bool
	}{
		{"identical", reflect.TypeFor[string](), reflect.TypeFor[string](), true},
		{"same underlying string enums", reflect.TypeFor[copierKindA](), reflect.TypeFor[copierKindB](), true},
		{"int to string", reflect.TypeFor[string](), reflect.TypeFor[int](), false},
		{"int64 to int32", reflect.TypeFor[int32](), reflect.TypeFor[int64](), false},
		{"bytes to string", reflect.TypeFor[string](), reflect.TypeFor[[]byte](), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, safeConvert(tt.dst, tt.src))
		})
	}
}

// fillValue sets every exported field to a non-zero value, bounded by depth to avoid cycles.
// Each struct's ForceSendFields is populated with its own field names so the FSF-filtering
// path is exercised at every level.
func fillValue(v reflect.Value, depth int, visited map[reflect.Type]bool) {
	if depth > 5 {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillValue(v.Elem(), depth+1, visited)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fillValue(s.Index(0), depth+1, visited)
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		key := reflect.New(v.Type().Key()).Elem()
		fillValue(key, depth+1, visited)
		val := reflect.New(v.Type().Elem()).Elem()
		fillValue(val, depth+1, visited)
		m.SetMapIndex(key, val)
		v.Set(m)
	case reflect.Struct:
		if visited[v.Type()] {
			return
		}
		visited[v.Type()] = true
		defer delete(visited, v.Type())
		var names []string
		for i := range v.NumField() {
			sf := v.Type().Field(i)
			if sf.PkgPath != "" || sf.Name == "ForceSendFields" {
				continue
			}
			names = append(names, sf.Name)
			fillValue(v.Field(i), depth+1, visited)
		}
		if f := v.FieldByName("ForceSendFields"); f.IsValid() && f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
			f.Set(reflect.ValueOf(names))
		}
	}
}
