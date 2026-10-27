package dyn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDropKeysTest(t *testing.T) {
	v := V(map[string]Value{
		"key1": V("value1"),
		"key2": V("value2"),
		"key3": V("value3"),
	})

	vout, err := DropKeys(v, []string{"key1", "key3"})
	require.NoError(t, err)

	mv := vout.MustMap()
	require.Equal(t, 1, mv.Len())
	v, ok := mv.GetByString("key2")
	require.True(t, ok)
	require.Equal(t, "value2", v.MustString())
}
