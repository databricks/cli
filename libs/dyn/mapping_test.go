package dyn_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestNewMapping(t *testing.T) {
	m := dyn.NewMapping()
	assert.Equal(t, 0, m.Len())
}

func TestMappingUninitialized(t *testing.T) {
	var m dyn.Mapping
	assert.Equal(t, 0, m.Len())

	value, ok := m.Get(dyn.V("key"))
	assert.Equal(t, dyn.InvalidValue, value)
	assert.False(t, ok)
	assert.Len(t, m.Keys(), 0)
	assert.Len(t, m.Values(), 0)
}

func TestMappingGetSet(t *testing.T) {
	var err error

	// Set a value
	var m dyn.Mapping
	err = m.Set(dyn.V("key1"), dyn.V("foo"))
	assert.NoError(t, err)
	assert.Equal(t, 1, m.Len())

	value, ok := m.Get(dyn.V("key1"))
	assert.Equal(t, dyn.V("foo"), value)
	assert.True(t, ok)

	value, ok = m.Get(dyn.V("invalid1"))
	assert.Equal(t, dyn.InvalidValue, value)
	assert.False(t, ok)

	// Set another value
	err = m.Set(dyn.V("key2"), dyn.V("bar"))
	assert.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	value, ok = m.Get(dyn.V("key2"))
	assert.Equal(t, dyn.V("bar"), value)
	assert.True(t, ok)

	value, ok = m.Get(dyn.V("invalid2"))
	assert.Equal(t, dyn.InvalidValue, value)
	assert.False(t, ok)

	// Overwrite first value
	err = m.Set(dyn.V("key1"), dyn.V("qux"))
	assert.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	value, ok = m.Get(dyn.V("key1"))
	assert.Equal(t, dyn.V("qux"), value)
	assert.True(t, ok)

	value, ok = m.Get(dyn.V("key2"))
	assert.Equal(t, dyn.V("bar"), value)
	assert.True(t, ok)

	// Try to set non-string key
	err = m.Set(dyn.V(1), dyn.V("qux"))
	assert.Error(t, err)
	assert.Equal(t, 2, m.Len())
}

func TestMappingKeysValues(t *testing.T) {
	var err error

	// Configure mapping
	var m dyn.Mapping
	err = m.Set(dyn.V("key1"), dyn.V("foo"))
	assert.NoError(t, err)
	err = m.Set(dyn.V("key2"), dyn.V("bar"))
	assert.NoError(t, err)

	// Confirm keys
	keys := m.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, dyn.V("key1"))
	assert.Contains(t, keys, dyn.V("key2"))

	// Confirm values
	values := m.Values()
	assert.Len(t, values, 2)
	assert.Contains(t, values, dyn.V("foo"))
	assert.Contains(t, values, dyn.V("bar"))
}

func TestMappingMerge(t *testing.T) {
	var m1 dyn.Mapping
	for i := 0; i < 10; i++ {
		err := m1.Set(dyn.V(fmt.Sprintf("%d", i)), dyn.V(i))
		require.NoError(t, err)
	}

	var m2 dyn.Mapping
	for i := 5; i < 15; i++ {
		err := m2.Set(dyn.V(fmt.Sprintf("%d", i)), dyn.V(i))
		require.NoError(t, err)
	}

	var out dyn.Mapping
	out.Merge(m1)
	assert.Equal(t, 10, out.Len())
	out.Merge(m2)
	assert.Equal(t, 15, out.Len())
}
