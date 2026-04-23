package variable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariable_IsComplexValued(t *testing.T) {
	tests := []struct {
		name string
		v    *Variable
		want bool
	}{
		{
			name: "map",
			v:    &Variable{Value: map[string]any{"foo": "bar"}},
			want: true,
		},
		{
			name: "slice",
			v:    &Variable{Value: []any{1, 2, 3}},
			want: true,
		},
		{
			name: "struct",
			v:    &Variable{Value: struct{ Foo string }{Foo: "bar"}},
			want: true,
		},
		{
			name: "scalar",
			v:    &Variable{Value: "foo"},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.v.IsComplexValued())
		})
	}
}

func TestVariable_SetAndHas(t *testing.T) {
	v := &Variable{Default: "d"}
	assert.True(t, v.HasDefault())
	assert.False(t, v.HasValue())

	require.NoError(t, v.Set("x"))
	assert.True(t, v.HasValue())
	assert.Equal(t, "x", v.Value)

	err := v.Set("y")
	assert.ErrorContains(t, err, "already been assigned")
}

func TestVariable_SetAllowsAnyIncomingShape(t *testing.T) {
	// Matches bundle.variable parity: Set checks .Value, not val, so any
	// shape is accepted on first assignment. Schema validation happens
	// elsewhere.
	v := &Variable{}
	require.NoError(t, v.Set(map[string]any{"k": "v"}))
	assert.True(t, v.IsComplexValued())
}
