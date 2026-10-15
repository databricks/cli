package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

type pointerScalarEnum string

type pointerScalars struct {
	S   *string            `json:"s,omitempty"`
	I   *int               `json:"i,omitempty"`
	I64 *int64             `json:"i64,omitempty"`
	B   *bool              `json:"b,omitempty"`
	F   *float64           `json:"f,omitempty"`
	E   pointerScalarEnum  `json:"e,omitempty"`
	N   *pointerScalarsSub `json:"n,omitempty"`
}

type pointerScalarsSub struct {
	B *bool `json:"b,omitempty"`
}

func TestFromTypedPointerScalarsNil(t *testing.T) {
	nv, err := FromTyped(pointerScalars{}, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedPointerScalarsZero(t *testing.T) {
	src := pointerScalars{
		S:   new(""),
		I:   new(0),
		I64: new(int64(0)),
		B:   new(false),
		F:   new(0.0),
		N:   &pointerScalarsSub{B: new(false)},
	}
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"s":   dyn.V(""),
		"i":   dyn.V(int64(0)),
		"i64": dyn.V(int64(0)),
		"b":   dyn.V(false),
		"f":   dyn.V(0.0),
		"n":   dyn.V(map[string]dyn.Value{"b": dyn.V(false)}),
	}), nv)
}

func TestFromTypedPointerScalarsSet(t *testing.T) {
	src := pointerScalars{S: new("x"), I: new(3), B: new(true), E: "A"}
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"s": dyn.V("x"),
		"i": dyn.V(int64(3)),
		"b": dyn.V(true),
		"e": dyn.V("A"),
	}), nv)
}

func TestFromTypedPointerScalarsClearedField(t *testing.T) {
	// A field present in the reference but now nil is dropped.
	ref := dyn.V(map[string]dyn.Value{"s": dyn.V("old"), "b": dyn.V(true)})
	nv, err := FromTyped(pointerScalars{B: new(false)}, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{"b": dyn.V(false)}), nv)
}

func TestToTypedPointerScalars(t *testing.T) {
	var out pointerScalars
	err := ToTyped(&out, dyn.V(map[string]dyn.Value{
		"s":   dyn.V(""),
		"i":   dyn.V(int64(0)),
		"i64": dyn.V(int64(5)),
		"b":   dyn.V(false),
		"f":   dyn.V(0.0),
		"e":   dyn.V("B"),
		"n":   dyn.V(map[string]dyn.Value{"b": dyn.V(true)}),
	}))
	require.NoError(t, err)
	assert.Equal(t, pointerScalars{
		S:   new(""),
		I:   new(0),
		I64: new(int64(5)),
		B:   new(false),
		F:   new(0.0),
		E:   "B",
		N:   &pointerScalarsSub{B: new(true)},
	}, out)
}

func TestToTypedPointerScalarsMissing(t *testing.T) {
	out := pointerScalars{S: new("previous")}
	err := ToTyped(&out, dyn.V(map[string]dyn.Value{"b": dyn.V(true)}))
	require.NoError(t, err)
	assert.Equal(t, pointerScalars{B: new(true)}, out)
}

func TestPointerScalarsRoundTrip(t *testing.T) {
	src := pointerScalars{S: new(""), B: new(false), I: new(0)}
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	var out pointerScalars
	require.NoError(t, ToTyped(&out, nv))
	assert.Equal(t, src, out)
}

func TestNormalizePointerScalars(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"s": dyn.V(""),
		"i": dyn.V("7"),
		"b": dyn.V(false),
		"e": dyn.V("C"),
	})
	vout, diags := Normalize(pointerScalars{}, vin)
	require.Empty(t, diags)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"s": dyn.V(""),
		"i": dyn.V(int64(7)),
		"b": dyn.V(false),
		"e": dyn.V("C"),
	}), vout)
}
