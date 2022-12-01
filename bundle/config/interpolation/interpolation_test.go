package interpolation

import (
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
