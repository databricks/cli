package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestNewRefNoString(t *testing.T) {
	_, ok := newRef(dyn.V(1))
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
		ref, ok := newRef(dyn.V(in))
		require.True(t, ok, "should match valid pattern: %s", in)
		assert.Equal(t, refs, ref.references())
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
		_, ok := newRef(dyn.V(v))
		require.False(t, ok, "should not match invalid pattern: %s", v)
	}
}

func TestIsPureVariableReference(t *testing.T) {
	assert.False(t, IsPureVariableReference(""))
	assert.False(t, IsPureVariableReference("${foo.bar} suffix"))
	assert.False(t, IsPureVariableReference("prefix ${foo.bar}"))
	assert.True(t, IsPureVariableReference("${foo.bar}"))
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
