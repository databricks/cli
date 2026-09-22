package dresources

import (
	"reflect"
	"slices"
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
		fillNonZero(remote.Elem(), 0)
		setForceSendFields(remote.Elem())

		want := m.Call([]reflect.Value{remote})[0].Interface()
		got := copier.copy(remote.Interface())
		if reflect.DeepEqual(want, got) {
			redundant = append(redundant, resourceType)
		}
	}

	slices.Sort(redundant)
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
		{
			name: "identical",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[string](),
			want: true,
		},
		{
			name: "same underlying string enums",
			dst:  reflect.TypeFor[copierKindA](),
			src:  reflect.TypeFor[copierKindB](),
			want: true,
		},
		{
			name: "int to string",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[int](),
			want: false,
		},
		{
			name: "int64 to int32",
			dst:  reflect.TypeFor[int32](),
			src:  reflect.TypeFor[int64](),
			want: false,
		},
		{
			name: "bytes to string",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[[]byte](),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, safeConvert(tt.dst, tt.src))
		})
	}
}

// setForceSendFields recursively sets every struct's ForceSendFields to its own field names,
// so the copier's ForceSendFields-filtering path is exercised at every level. fillNonZero
// deliberately leaves ForceSendFields empty (round-trip tests need that), so we populate it
// here. Map values are not addressable and cannot hold a relevant FSF source, so they are skipped.
func setForceSendFields(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			setForceSendFields(v.Elem())
		}
	case reflect.Slice:
		for i := range v.Len() {
			setForceSendFields(v.Index(i))
		}
	case reflect.Struct:
		var names []string
		for i := range v.NumField() {
			sf := v.Type().Field(i)
			if !sf.IsExported() || sf.Name == "ForceSendFields" {
				continue
			}
			names = append(names, sf.Name)
			setForceSendFields(v.Field(i))
		}
		if f := v.FieldByName("ForceSendFields"); f.IsValid() && f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
			f.Set(reflect.ValueOf(names))
		}
	default:
		// scalars and other kinds have no nested ForceSendFields to set
	}
}
