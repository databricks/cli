package interpolation

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
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

func TestInterpolationWithResursiveVariableReferences(t *testing.T) {
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
	assert.ErrorContains(t, err, "cycle detected in field resolution: b -> c -> d -> b")
}

func TestInterpolationForVariables(t *testing.T) {
	foo := "abc"
	bar := "${var.foo} def"
	apple := "${var.foo} ${var.bar}"
	config := config.Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Value: &foo,
			},
			"bar": {
				Value: &bar,
			},
			"apple": {
				Value: &apple,
			},
		},
		Bundle: config.Bundle{
			Name: "${var.apple} ${var.foo}",
		},
	}

	err := expand(&config)
	assert.NoError(t, err)
	assert.Equal(t, "abc", *(config.Variables["foo"].Value))
	assert.Equal(t, "abc def", *(config.Variables["bar"].Value))
	assert.Equal(t, "abc abc def", *(config.Variables["apple"].Value))
	assert.Equal(t, "abc abc def abc", config.Bundle.Name)
}

func TestInterpolationLoopForVariables(t *testing.T) {
	foo := "${var.bar}"
	bar := "${var.foo}"
	config := config.Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Value: &foo,
			},
			"bar": {
				Value: &bar,
			},
		},
		Bundle: config.Bundle{
			Name: "${var.foo}",
		},
	}

	err := expand(&config)
	assert.ErrorContains(t, err, "cycle detected in field resolution: bundle.name -> var.foo -> var.bar -> var.foo")
}

func TestInterpolationInvalidVariableReference(t *testing.T) {
	foo := "abc"
	config := config.Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Value: &foo,
			},
		},
		Bundle: config.Bundle{
			Name: "${vars.foo}",
		},
	}

	err := expand(&config)
	assert.ErrorContains(t, err, "could not resolve reference vars.foo")
}
