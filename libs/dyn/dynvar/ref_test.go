package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefNoString(t *testing.T) {
	_, ok := newRef(dyn.V(1))
	require.False(t, ok, "should not match non-string")
}

func TestNewRefValidPattern(t *testing.T) {
	for in, refs := range map[string][]string{
		"${hello_world.world_world}": {"hello_world.world_world"},
		"${helloworld.world-world}":  {"helloworld.world-world"},
		"${hello-world.world-world}": {"hello-world.world-world"},
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
		"${a-a.a-_a-a.id}",              // fails because of -_ in the second segment
		"${a-a.a--a-a.id}",              // fails because of -- in the second segment
	}
	for _, v := range invalid {
		_, ok := newRef(dyn.V(v))
		require.False(t, ok, "should not match invalid pattern: %s", v)
	}
}
