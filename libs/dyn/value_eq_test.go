package dyn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueEqMap(t *testing.T) {
	loc := []Location{{File: "file", Line: 1, Column: 2}}
	v := NewValue(map[string]Value{"key": V("value")}, loc)

	tests := []struct {
		name string
		a, b Value
		want bool
	}{
		{
			name: "same underlying mapping",
			a:    v,
			b:    v,
			want: true,
		},
		{
			name: "cloned mapping with equal contents",
			a:    v,
			b:    Value{v: v.v.(Mapping).Clone(), k: KindMap, l: v.l},
			want: false,
		},
		{
			name: "different lengths",
			a:    v,
			b:    NewValue(map[string]Value{"key": V("value"), "other": V("value")}, loc),
			want: false,
		},
		{
			name: "different locations",
			a:    v,
			b:    v.WithLocations([]Location{{File: "other", Line: 1, Column: 2}}),
			want: false,
		},
		{
			name: "empty mappings",
			a:    NewValue(map[string]Value{}, loc),
			b:    NewValue(map[string]Value{}, loc),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.a.eq(tc.b))
		})
	}
}

func TestValueEqMapVisitDoesNotRebuildAncestors(t *testing.T) {
	vin := V(map[string]Value{
		"a": V(map[string]Value{
			"b": V("value"),
		}),
	})

	vout, err := Map(vin, "a.b", func(_ Path, v Value) (Value, error) {
		return v, nil
	})
	require.NoError(t, err)

	// The identity transform must return the original value without
	// cloning the ancestor maps.
	vm := vin.v.(Mapping)
	wm := vout.v.(Mapping)
	require.Equal(t, vm.Len(), wm.Len())
	assert.Same(t, &vm.pairs[0], &wm.pairs[0])
}
