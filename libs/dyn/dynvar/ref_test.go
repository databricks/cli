package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefNoString(t *testing.T) {
	_, ok := NewRef(dyn.V(1))
	require.False(t, ok, "should not match non-string")
}

func TestNewRefValidPattern(t *testing.T) {
	for in, refs := range map[string][]string{
		"${hello_world.world_world}":  {"hello_world.world_world"},
		"${helloworld.world-world}":   {"helloworld.world-world"},
		"${hello-world.world-world}":  {"hello-world.world-world"},
		"${hello_world.world__world}": {"hello_world.world__world"},
		"${hello_world.world--world}": {"hello_world.world--world"},
		"${hello_world.world-_world}": {"hello_world.world-_world"},
		"${hello_world.world_-world}": {"hello_world.world_-world"},
	} {
		ref, ok := NewRef(dyn.V(in))
		require.True(t, ok, "should match valid pattern: %s", in)
		assert.Equal(t, refs, ref.References())
	}
}

func TestNewRefInvalidPattern(t *testing.T) {
	invalid := []string{
		"${hello_world-.world_world}",   // the first segment ending must not end with hyphen (-)
		"${hello_world-_.world_world}",  // the first segment ending must not end with underscore (_)
		"${helloworld.world-world-}",    // second segment must not end with hyphen (-)
		"${helloworld-.world-world}",    // first segment must not end with hyphen (-)
		"${helloworld.-world-world}",    // second segment must not start with hyphen (-)
		"${-hello-world.-world-world-}", // must not start or end with hyphen (-)
		"${_-_._-_.id}",                 // cannot use _- in sequence
		"${0helloworld.world-world}",    // interpolated first section shouldn't start with number
		"${helloworld.9world-world}",    // interpolated second section shouldn't start with number
	}
	for _, v := range invalid {
		_, ok := NewRef(dyn.V(v))
		require.False(t, ok, "should not match invalid pattern: %s", v)
	}
}

func TestIsPureVariableReference(t *testing.T) {
	assert.False(t, IsPureVariableReference(""))
	assert.False(t, IsPureVariableReference("${foo.bar} suffix"))
	assert.False(t, IsPureVariableReference("prefix ${foo.bar}"))
	assert.True(t, IsPureVariableReference("${foo.bar}"))
}

func TestFindAllInterpolationReferences(t *testing.T) {
	tests := []struct {
		input    string
		expected []InterpolationReference
	}{
		{
			input:    "no interpolation",
			expected: nil,
		},
		{
			input: "${var.foo}",
			expected: []InterpolationReference{
				{Match: "${var.foo}", Path: "var.foo"},
			},
		},
		{
			input: "echo ${bundle.name} and ${workspace.host}",
			expected: []InterpolationReference{
				{Match: "${bundle.name}", Path: "bundle.name"},
				{Match: "${workspace.host}", Path: "workspace.host"},
			},
		},
		{
			input: "${resources.jobs.my_job.id}",
			expected: []InterpolationReference{
				{Match: "${resources.jobs.my_job.id}", Path: "resources.jobs.my_job.id"},
			},
		},
		{
			input: "${FOO}",
			expected: []InterpolationReference{
				{Match: "${FOO}", Path: "FOO"},
			},
		},
		{
			// Bash parameter expansion doesn't match DAB regex
			input:    "${VAR:-default}",
			expected: nil,
		},
		{
			// Plain bash variable without braces
			input:    "$FOO",
			expected: nil,
		},
		{
			input: "${variables.my_var.value}",
			expected: []InterpolationReference{
				{Match: "${variables.my_var.value}", Path: "variables.my_var.value"},
			},
		},
	}

	for _, tc := range tests {
		refs := FindAllInterpolationReferences(tc.input)
		assert.Equal(t, tc.expected, refs, "input: %s", tc.input)
	}
}

func TestHasValidDABPrefix(t *testing.T) {
	// Valid DAB prefixes
	assert.True(t, HasValidDABPrefix("var.foo"))
	assert.True(t, HasValidDABPrefix("bundle.name"))
	assert.True(t, HasValidDABPrefix("workspace.host"))
	assert.True(t, HasValidDABPrefix("variables.my_var.value"))
	assert.True(t, HasValidDABPrefix("resources.jobs.my_job.id"))
	assert.True(t, HasValidDABPrefix("artifacts.my_artifact.path"))

	// Prefix alone (edge case - valid but may not resolve to anything useful)
	assert.True(t, HasValidDABPrefix("var"))
	assert.True(t, HasValidDABPrefix("bundle"))

	// Invalid prefixes (not known DAB prefixes)
	assert.False(t, HasValidDABPrefix("FOO"))
	assert.False(t, HasValidDABPrefix("MY_VAR"))
	assert.False(t, HasValidDABPrefix("unknown.path"))
	assert.False(t, HasValidDABPrefix("env.VAR"))
	assert.False(t, HasValidDABPrefix(""))
}

func TestPureReferenceToPath(t *testing.T) {
	for _, tc := range []struct {
		in  string
		out string
		ok  bool
	}{
		{"${foo.bar}", "foo.bar", true},
		{"${foo.bar.baz}", "foo.bar.baz", true},
		{"${foo.bar.baz[0]}", "foo.bar.baz[0]", true},
		{"${foo.bar.baz[0][1]}", "foo.bar.baz[0][1]", true},
		{"${foo.bar.baz[0][1].qux}", "foo.bar.baz[0][1].qux", true},

		{"${foo.one}${foo.two}", "", false},
		{"prefix ${foo.bar}", "", false},
		{"${foo.bar} suffix", "", false},
		{"${foo.bar", "", false},
		{"foo.bar}", "", false},
		{"foo.bar", "", false},
		{"{foo.bar}", "", false},
		{"", "", false},
	} {
		path, ok := PureReferenceToPath(tc.in)
		if tc.ok {
			assert.True(t, ok)
			assert.Equal(t, dyn.MustPathFromString(tc.out), path)
		} else {
			assert.False(t, ok)
		}
	}
}
