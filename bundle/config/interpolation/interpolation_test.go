package interpolation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nest struct {
	X string            `json:"x"`
	Y *string           `json:"y"`
	Z map[string]string `json:"z"`
}

type foo struct {
	A string `json:"a"`
	B string `json:"b"`
	C string `json:"c"`

	// Pointer field
	D *string `json:"d"`

	// Struct field
	E nest `json:"e"`

	// Map field
	F map[string]string `json:"f"`
}

func expand(v any) error {
	a := accumulator{}
	a.start(v)
	return a.expand(DefaultLookup)
}

func TestInterpolationVariables(t *testing.T) {
	f := foo{
		A: "a",
		B: "${a}",
		C: "${a}",
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "a", f.A)
	assert.Equal(t, "a", f.B)
	assert.Equal(t, "a", f.C)
}

func TestInterpolationWithPointers(t *testing.T) {
	fd := "${a}"
	f := foo{
		A: "a",
		D: &fd,
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "a", f.A)
	assert.Equal(t, "a", *f.D)
}

func TestInterpolationWithStruct(t *testing.T) {
	fy := "${e.x}"
	f := foo{
		A: "${e.x}",
		E: nest{
			X: "x",
			Y: &fy,
		},
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "x", f.A)
	assert.Equal(t, "x", f.E.X)
	assert.Equal(t, "x", *f.E.Y)
}

func TestInterpolationWithMap(t *testing.T) {
	f := foo{
		A: "${f.a}",
		F: map[string]string{
			"a": "a",
			"b": "${f.a}",
		},
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "a", f.A)
	assert.Equal(t, "a", f.F["a"])
	assert.Equal(t, "a", f.F["b"])
}

func TestRecursiveInterpolationVariables(t *testing.T) {
	f := foo{
		A: "a",
		B: "(${a})",
		C: "${a} ${b}",
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "a", f.A)
	assert.Equal(t, "(a)", f.B)
	assert.Equal(t, "a (a)", f.C)
}

func TestInterpolationVariableLoopError(t *testing.T) {
	d := "${b}"
	f := foo{
		A: "a",
		B: "${c}",
		C: "${d}",
		D: &d,
	}

	err := expand(&f)
	assert.ErrorContains(t, err, "cycle retected in field resolution:")

	// could be all possiblities since map traversal in golang is randomized
	assert.True(t, strings.Contains(err.Error(), "b -> c -> d -> b") ||
			strings.Contains(err.Error(), "c -> d -> b -> c") ||
			strings.Contains(err.Error(), "d -> b -> c -> d"))
}
