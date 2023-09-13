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

func TestInterpolationVariablesSpecialChars(t *testing.T) {
	type bar struct {
		A string `json:"a-b"`
		B string `json:"b_c"`
		C string `json:"c-_a"`
	}
	f := bar{
		A: "a",
		B: "${a-b}",
		C: "${a-b}",
	}

	err := expand(&f)
	require.NoError(t, err)

	assert.Equal(t, "a", f.A)
	assert.Equal(t, "a", f.B)
	assert.Equal(t, "a", f.C)
}

func TestInterpolationValidMatches(t *testing.T) {
	expectedMatches := map[string]string{
		"${hello_world.world_world}": "hello_world.world_world",
		"${helloworld.world-world}":  "helloworld.world-world",
		"${hello-world.world-world}": "hello-world.world-world",
	}
	for interpolationStr, expectedMatch := range expectedMatches {
		match := re.FindStringSubmatch(interpolationStr)
		assert.True(t, len(match) > 0,
			"Failed to match %s and find %s", interpolationStr, expectedMatch)
		assert.Equal(t, expectedMatch, match[1],
			"Failed to match the exact pattern %s and find %s", interpolationStr, expectedMatch)
	}
}

func TestInterpolationInvalidMatches(t *testing.T) {
	invalidMatches := []string{
		"${hello_world-.world_world}",   // the first segment ending must not end with hyphen (-)
		"${hello_world-_.world_world}",  // the first segment ending must not end with underscore (_)
		"${helloworld.world-world-}",    // second segment must not end with hyphen (-)
		"${helloworld-.world-world}",    // first segment must not end with hyphen (-)
		"${helloworld.-world-world}",    // second segment must not start with hyphen (-)
		"${-hello-world.-world-world-}", // must not start or end with hyphen (-)
		"${_-_._-_.id}",                 // cannot use _- in sequence
		"${0helloworld.world-world}",    // interpolated first section shouldn't start with number
		"${helloworld.9world-world}",    // interpolated second section shouldn't start with number
		"${a-a.a-_a-a.id}",              // fails because of -_ in the second segment
		"${a-a.a--a-a.id}",              // fails because of -- in the second segment
	}
	for _, invalidMatch := range invalidMatches {
		match := re.FindStringSubmatch(invalidMatch)
		assert.True(t, len(match) == 0, "Should be invalid interpolation: %s", invalidMatch)
	}
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
	assert.ErrorContains(t, err, "no value found for interpolation reference: ${vars.foo}")
}
